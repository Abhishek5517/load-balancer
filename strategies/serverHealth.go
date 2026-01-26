package strategies

// example
var serverHealth = []bool{true, true}

func HealthCheckServer() string {
	for i := 0; i < TotalServer; i++ {
		counter = (counter + 1) % TotalServer
		if serverHealth[counter] {
			return GetUrl(counter)
		}
	}
	return "" // No healthy servers available
}
