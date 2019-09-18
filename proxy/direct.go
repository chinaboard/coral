package proxy

import (
	"net"
	"time"
)

type DirectProxy struct {
	Timeout time.Duration
}

func NewDirectProxy(timeout time.Duration) *DirectProxy {
	return &DirectProxy{Timeout: timeout}
}

func (this *DirectProxy) Dial(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, this.Timeout)
}

func (this *DirectProxy) Name() string {
	return "DIRECT"
}
