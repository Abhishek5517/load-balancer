package myRedis

import (
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

func InitRedis() redis.Client {
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatalf("ParseURL failed: %v", err)
	}
	return *redis.NewClient(opt)
}
