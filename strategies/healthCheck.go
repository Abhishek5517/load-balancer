package strategies

import (
	"log"
	"net/http"
	"time"
)

// StartHealthChecker pings every registered server's /health endpoint
// at the given interval and updates their health status in the pool.
func StartHealthChecker(pool *Pool, interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			for _, s := range pool.List() {
				go pingServer(pool, s.URL)
			}
		}
	}()
}

func pingServer(pool *Pool, url string) {
	resp, err := http.Get(url + "/health")
	if err != nil || resp.StatusCode >= 400 {
		pool.SetHealth(url, false)
		log.Printf("[health] %s is DOWN", url)
		return
	}
	pool.SetHealth(url, true)
	log.Printf("[health] %s is UP", url)
}
