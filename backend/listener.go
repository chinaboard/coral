package backend

import (
	"net/http"

	"github.com/chinaboard/coral/backend/proxy"
)

type SelectProxyFunc func(req *http.Request, proxies []proxy.Proxy, domestic bool) (proxy.Proxy, error)

type Listener interface {
	ListenAndServe() error
	RegisterProxy(proxy proxy.Proxy) (bool, error)
	RegisterLoadBalance(SelectProxyFunc) (bool, error)
}
