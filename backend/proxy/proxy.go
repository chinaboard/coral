package proxy

import (
	"net"
	"time"
)

type Proxy interface {
	Dial(string) (net.Conn, time.Duration, error)
	Name() string
	Domestic() bool
}
