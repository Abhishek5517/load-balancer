package strategies

import "sync/atomic"

type Server struct {
	URL               string
	Healthy           bool
	ActiveConnections atomic.Int32
}
