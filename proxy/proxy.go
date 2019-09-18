package proxy

import (
	"coral/config"
	"net"

	"github.com/juju/errors"
)

type DialFunc func(addr string) (net.Conn, error)

type Proxy interface {
	Dial(string) (net.Conn, error)
	Name() string
}

func GenProxy(server config.CoralServer) (Proxy, error) {

	switch server.Type {
	case "ss":
		return NewShadowsocksProxy(server)
	case "ssr":
		return NewShadowsocksRProxy(server)
	default:
		return nil, errors.NotSupportedf(server.Type)
	}

	return nil, errors.NotFoundf("Address not found")
}
