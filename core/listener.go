package core

import (
	"net/http"

	"github.com/chinaboard/coral/core/proxy"
)

type SelectProxyFunc func(req *http.Request, proxies []proxy.Proxy, direct bool) (proxy.Proxy, error)

type Listener interface {
	ListenAndServe() error
	RegisterProxy(proxy proxy.Proxy) (bool, error)
	RegisterLoadBalance(SelectProxyFunc) (bool, error)
}
