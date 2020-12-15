package proxy

import (
	"net"
	"time"
)

type Proxy interface {
	Dial(network, addr string) (net.Conn, time.Duration, error)
	Name() string
	Direct() bool
}
