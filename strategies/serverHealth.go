package strategies

func HealthCheckServer() (string, error) {
	return DefaultPool.Next()
}
