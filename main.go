package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Abhishek5517/load-balancer/strategies"

	myRedis "github.com/Abhishek5517/load-balancer/RedisClient"

	ratelimiter "github.com/Abhishek5517/Rate-Limiter"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

var (
	_        = godotenv.Load()
	Redis    = myRedis.InitRedis()
	PORT     = os.Getenv("PORT")
	STRATEGY = os.Getenv("STRATEGY") // round-robin | least-connection | consistent-hashing | random-server
)

func main() {
	app := fiber.New()

	strategies.StartHealthChecker(strategies.DefaultPool, 10*time.Second)

	app.Post("/register", RegisterServer())
	app.Post("/deregister", DeregisterServer())
	app.Get("/servers", ListServers())
	app.All("*", ClientRequest())

	log.Printf("[config] strategy: %s", strategyName())
	log.Fatal(app.Listen(PORT))
}

func strategyName() string {
	switch STRATEGY {
	case "least-connection", "consistent-hashing", "random-server":
		return STRATEGY
	default:
		return "round-robin"
	}
}

type serverRequest struct {
	URL string `json:"url"`
}

func RegisterServer() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body serverRequest
		if err := c.BodyParser(&body); err != nil || body.URL == "" {
			return c.Status(400).SendString("request body must be JSON with a 'url' field")
		}
		strategies.DefaultPool.Register(body.URL)
		log.Printf("[registry] registered: %s", body.URL)
		return c.SendStatus(200)
	}
}

func DeregisterServer() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body serverRequest
		if err := c.BodyParser(&body); err != nil || body.URL == "" {
			return c.Status(400).SendString("request body must be JSON with a 'url' field")
		}
		strategies.DefaultPool.Deregister(body.URL)
		log.Printf("[registry] deregistered: %s", body.URL)
		return c.SendStatus(200)
	}
}

func ListServers() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(strategies.DefaultPool.List())
	}
}

func ClientRequest() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		// token bucket rate limiter: 1 request/sec per IP, burst of 1
		if !ratelimiter.RateLimit(&Redis, ip, 1, 1, 1) {
			return c.Status(429).SendString("Too Many Requests")
		}

		targetOrigin, err := selectServer(ip)
		if err != nil {
			return c.Status(503).SendString("No healthy servers available")
		}

		// track active connections for least-connection strategy
		if STRATEGY == "least-connection" {
			strategies.DefaultPool.IncrementConn(targetOrigin)
		}

		targetURL := targetOrigin + c.OriginalURL()

		req, err := http.NewRequest(c.Method(), targetURL, bytes.NewReader(c.Body()))
		if err != nil {
			return c.Status(500).SendString("Failed to create request: " + err.Error())
		}

		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		respChan := make(chan *http.Response)
		errChan := make(chan error)

		go func() {
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				// passive health check: mark server down immediately on failure
				strategies.DefaultPool.SetHealth(targetOrigin, false)
				log.Printf("[health] %s marked DOWN (proxy failure)", targetOrigin)
				errChan <- err
				return
			}
			respChan <- resp
		}()

		select {
		case resp := <-respChan:
			defer resp.Body.Close()
			if STRATEGY == "least-connection" {
				strategies.DefaultPool.DecrementConn(targetOrigin)
			}
			c.Set("Content-Type", resp.Header.Get("Content-Type"))
			c.Status(resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			return c.Send(body)

		case err := <-errChan:
			if STRATEGY == "least-connection" {
				strategies.DefaultPool.DecrementConn(targetOrigin)
			}
			return c.Status(500).SendString("Failed to forward request: " + err.Error())
		}
	}
}

// selectServer picks the target backend based on the configured strategy.
func selectServer(clientIP string) (string, error) {
	switch STRATEGY {
	case "least-connection":
		return strategies.LeastConnectionServer()
	case "consistent-hashing":
		return strategies.ConsistentHashServer(clientIP)
	case "random-server":
		return strategies.RandomServer()
	default:
		return strategies.RoundRobinServer()
	}
}
