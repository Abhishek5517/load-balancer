package strategies

func RoundRobinServer() (string, error) {
	return DefaultPool.Next()
}
