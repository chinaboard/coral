package core

import (
	"github.com/chinaboard/coral/config"
	"github.com/chinaboard/coral/core/proxy"
	"github.com/chinaboard/coral/core/ss"
	"github.com/chinaboard/coral/core/ssr"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
)

func GenerateProxy(server config.CoralServer) (proxy.Proxy, error) {
	log.Infoln("init", server.Type, server.Name, server.Address(), "...")
	switch server.Type {
	case "ss":
		return ss.New(server)
	case "ssr":
		return ssr.New(server)
	default:
		return nil, errors.NotSupportedf(server.Type)
	}
}
