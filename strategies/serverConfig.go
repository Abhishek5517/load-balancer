package strategies

type Server struct {
	URL          string
	ServerNumber int
}

var mp = make(map[int]string)

// temporary function to get url
func GetUrl(counter int) string {
	mp[0] = "http://localhost:9000"
	mp[1] = "http://localhost:3000"

	return mp[counter]

}
