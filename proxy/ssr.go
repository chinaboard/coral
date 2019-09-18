package proxy

import (
	"coral/config"
	"fmt"
	"net"
	"net/url"
	"time"

	shadowsocksr "github.com/chinaboard/shadowsocksR"
	"github.com/juju/errors"

	"github.com/chinaboard/shadowsocksR/tools/socks"
)

type ShadowsocksRProxy struct {
	name         string
	Timeout      time.Duration
	Address      *url.URL
	ObfsData     interface{}
	ProtocolData interface{}
}

func NewShadowsocksRProxy(server config.CoralServer) (Proxy, error) {
	u := &url.URL{
		Scheme: server.Type,
		Host:   server.Address(),
	}
	v := u.Query()
	v.Set("encrypt-method", server.Method)
	v.Set("encrypt-key", server.Password)
	v.Set("obfs", server.Obfs)
	v.Set("obfs-param", server.ObfsParam)
	v.Set("protocol", server.Protocol)
	v.Set("protocol-param", server.ProtocolParam)
	u.RawQuery = v.Encode()

	return &ShadowsocksRProxy{
		name:    server.Name,
		Timeout: server.ReadTimeout,
		Address: u,
	}, nil
}

func (this *ShadowsocksRProxy) Dial(addr string) (net.Conn, error) {
	ssrconn, err := shadowsocksr.NewSSRClient(this.Address)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("connecting to SSR server failed :%v", err))
	}

	if this.ObfsData == nil {
		this.ObfsData = ssrconn.IObfs.GetData()
	}
	ssrconn.IObfs.SetData(this.ObfsData)

	if this.ProtocolData == nil {
		this.ProtocolData = ssrconn.IProtocol.GetData()
	}
	ssrconn.IProtocol.SetData(this.ProtocolData)

	if _, err := ssrconn.Write(socks.ParseAddr(addr)); err != nil {
		return nil, ssrconn.Close()
	}
	return ssrconn, nil
}

func (this *ShadowsocksRProxy) Name() string {
	return this.name
}
