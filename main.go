package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Abhishek5517/load-balancer/strategies"

	myRedis "github.com/Abhishek5517/load-balancer/RedisClient"

	ratelimiter "github.com/Abhishek5517/Rate-Limiter"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

var (
	_     = godotenv.Load()
	Redis = myRedis.InitRedis()
	PORT  = os.Getenv("PORT")
)

func main() {
	app := fiber.New()

	app.All("*", ClientRequest())

	log.Fatal(app.Listen(PORT))
}

func ClientRequest() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		// custom rate limiter: can be configured as per requirement ( requests, time window in seconds, burst)
		// token bucket algorithm
		// here it is set to 1 request per second with burst of 1
		// it can be used in scaled environment as well with redis as backend
		if !ratelimiter.RateLimit(&Redis, ip, 1, 1, 1) {
			return c.Status(429).SendString("Too Many Requests")
		}

		targetOrigin := strategies.RoundRobinServer()
		targetURL := targetOrigin + c.OriginalURL()

		// New http for comming request from client ( now from LB to target server)
		req, err := http.NewRequest(c.Method(), targetURL, bytes.NewReader(c.Body()))
		if err != nil {
			return c.Status(500).SendString("Failed to create request: " + err.Error())
		}

		// Copy headers from the original request
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		// Channel to receive the response or error
		respChan := make(chan *http.Response)
		errChan := make(chan error)

		// Perform the HTTP request in a goroutine to avoid blocking for different clients
		go func() {
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				errChan <- err
				return
			}
			respChan <- resp
		}()

		// using select to wait for either the response or an error
		select {
		case resp := <-respChan:
			defer resp.Body.Close()

			// Copy the response back to the Fiber context to send it to the client from LB back to client
			c.Set("Content-Type", resp.Header.Get("Content-Type"))
			c.Status(resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			return c.Send(body)

		case err := <-errChan:
			return c.Status(500).SendString("Failed to forward request: " + err.Error())
		}
	}
}
