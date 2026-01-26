package strategies

import (
	"math/rand"
)

func RandomServer() string {
	serverIndex := rand.Intn(TotalServer)
	return GetUrl(serverIndex)
}
