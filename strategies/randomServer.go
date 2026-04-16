package strategies

import (
	"errors"
	"math/rand"
)

func RandomServer() (string, error) {
	servers := DefaultPool.List()

	var healthy []string
	for _, s := range servers {
		if s.Healthy {
			healthy = append(healthy, s.URL)
		}
	}

	if len(healthy) == 0 {
		return "", errors.New("no healthy servers available")
	}

	return healthy[rand.Intn(len(healthy))], nil
}
