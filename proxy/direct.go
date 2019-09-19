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

func (this *DirectProxy) Dial(addr string) (net.Conn, time.Duration, error) {
	conn, err := net.DialTimeout("tcp", addr, this.Timeout)
	return conn, this.Timeout, err
}

func (this *DirectProxy) Name() string {
	return "DIRECT"
}
