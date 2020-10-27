package core

import (
	"github.com/chinaboard/coral/core/proxy"
)

type SelectProxyFunc func(addr string, proxies []proxy.Proxy, direct bool) (proxy.Proxy, error)

type Listener interface {
	ListenAndServe() error
	RegisterProxy(proxy.Proxy) (bool, error)
	RegisterLoadBalance(SelectProxyFunc) (bool, error)
	AuthIP(string) bool
	AuthUser(string, string) bool
}
