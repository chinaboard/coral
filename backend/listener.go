package backend

import "github.com/chinaboard/coral/backend/proxy"

type SelectProxyFunc func(addr string, proxies []proxy.Proxy, domestic bool) (proxy.Proxy, error)

type Listener interface {
	ListenAndServe() error
	RegisterProxy(proxy proxy.Proxy) (bool, error)
	RegisterLoadBalance(SelectProxyFunc) (bool, error)
}
