package strategies

const TotalServer int = 2

var counter int = 0

func RoundRobinServer() string {
	counter %= TotalServer
	url := GetUrl(counter)
	counter++
	return url
}
