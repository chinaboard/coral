package direct

import (
	"net"
	"time"

	"github.com/chinaboard/coral/core/proxy"
)

type DirectProxy struct {
	Timeout time.Duration
}

func New(timeout time.Duration) proxy.Proxy {
	return &DirectProxy{Timeout: timeout}
}

func (this *DirectProxy) Dial(addr string) (net.Conn, time.Duration, error) {
	conn, err := net.DialTimeout("tcp", addr, this.Timeout)
	return conn, this.Timeout, err
}

func (this *DirectProxy) Name() string {
	return "DIRECT"
}

func (this *DirectProxy) Direct() bool {
	return true
}
