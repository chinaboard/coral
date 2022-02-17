package core

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/chinaboard/coral/core/direct"

	"github.com/chinaboard/coral/core/proxy"

	"github.com/juju/errors"

	"github.com/chinaboard/coral/cache"
	"github.com/chinaboard/coral/config"
	"github.com/chinaboard/coral/leakybuf"
	log "github.com/sirupsen/logrus"
)

type httpListener struct {
	sync.Mutex
	cache           *cache.Cache
	proxies         []proxy.Proxy
	srv             *http.Server
	selectProxyFunc SelectProxyFunc
	whitelist       map[string]bool
}

func NewHttpListener(conf *config.CoralConfig) (Listener, error) {
	if conf == nil {
		return nil, errors.New("config is nil")
	}
	if len(conf.Servers) == 0 {
		return nil, errors.NotFoundf("server")
	}

	listener := &httpListener{
		proxies:   []proxy.Proxy{direct.New(conf.Common.DirectTimeout)},
		cache:     cache.NewCache(time.Minute * 30),
		whitelist: conf.Common.Whitelist,
	}

	listener.srv = &http.Server{
		Addr:    conf.Common.Address(),
		Handler: listener,
	}

	for _, v := range conf.Servers {
		p, err := GenerateProxy(v)
		if err != nil {
			log.Warningln(err)
			continue
		}
		if ok, err := listener.RegisterProxy(p); !ok {
			return nil, err
		}
	}

	if ok, err := listener.RegisterLoadBalance(listener.DefaultSelectProxy); !ok {
		return nil, err
	}

	return listener, nil
}

func (this *httpListener) ListenAndServe() error {
	if this.selectProxyFunc == nil {
		return errors.New("not found selectProxyFunc")
	}
	return this.srv.ListenAndServe()
}

func (this *httpListener) RegisterProxy(proxy proxy.Proxy) (bool, error) {
	if proxy != nil {
		this.Lock()
		defer this.Unlock()
		this.proxies = append(this.proxies, proxy)
		return true, nil
	}
	return false, errors.New("proxy is nil")
}

func (this *httpListener) RegisterLoadBalance(selectProxyFunc SelectProxyFunc) (bool, error) {
	if this.selectProxyFunc != nil {
		return false, errors.New("had func")
	}

	if selectProxyFunc != nil {
		this.selectProxyFunc = selectProxyFunc
		return true, nil
	}
	return false, errors.New("func errer")
}

func (this *httpListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			log.Debugf("panic: %v\n", err)
		}
	}()

	if !this.auth(w, r) {
		return
	}

	d := this.cache.ShouldDirect(r.Host)

	proxy, err := this.selectProxyFunc(r.Host, this.proxies, d)
	if err != nil {
		log.Errorln(err)
		return
	}
	log.Infoln(proxy.Name(), r.RemoteAddr, r.Method, r.Host)

	if r.Method == "CONNECT" {
		this.HandleConnect(w, r, proxy)
	} else {
		this.HandleHttp(w, r, proxy)
	}

}

func (this *httpListener) HandleConnect(w http.ResponseWriter, r *http.Request, proxy proxy.Proxy) {
	hj, _ := w.(http.Hijacker)
	lConn, _, err := hj.Hijack()
	if err != nil && err != http.ErrHijacked {
		log.Errorln("hijack", err)
		return
	}

	var (
		rConn   net.Conn
		timeout time.Duration
		errs    error
	)

	rConn, timeout, errs = proxy.Dial("tcp", r.Host)

	if errs != nil {
		log.Errorln(proxy.Name(), "Dial:", err, r.Host)
		return
	}
	lConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go this.Pipe(lConn, rConn, timeout)
	this.Pipe(rConn, lConn, timeout)
}

func (this *httpListener) HandleHttp(w http.ResponseWriter, r *http.Request, proxy proxy.Proxy) {
	tr := http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, _, err := proxy.Dial(network, addr)
			return conn, err
		},
	}

	r.Close = true
	resp, err := tr.RoundTrip(r)
	if err != nil {
		log.Errorln("request error: ", err, r.Host)
		return
	}
	defer resp.Body.Close()

	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func (this *httpListener) DefaultSelectProxy(addr string, proxies []proxy.Proxy, direct bool) (proxy.Proxy, error) {
	// first match proxy
	for _, value := range proxies {
		if direct == value.Direct() {
			return value, nil
		}
	}
	return nil, errors.NotFoundf("proxy: %s", direct)
}

func (this *httpListener) Pipe(src, dst net.Conn, timeout time.Duration) error {
	buf := leakybuf.GlobalLeakyBuf.Get()
	for {
		if timeout != 0 {
			src.SetReadDeadline(time.Now().Add(timeout))
		}
		n, err := src.Read(buf)
		// read may return EOF with n > 0
		// should always process n > 0 bytes before handling error
		if n > 0 {
			// Note: avoid overwrite err returned by Read.
			if _, err := dst.Write(buf[0:n]); err != nil {
				break
			}
		}
		if err != nil {
			// Always "use of closed network connection", but no easy way to
			// identify this specific error. So just leave the error along for now.
			// More info here: https://code.google.com/p/go/issues/detail?id=4373
			break
		}
	}
	leakybuf.GlobalLeakyBuf.Put(buf)
	dst.Close()
	return nil
}

func (this *httpListener) AuthUser(user, pwd string) bool {
	return true
}

func (this *httpListener) AuthIP(ip string) bool {
	if len(this.whitelist) > 1 {
		if _, ok := this.whitelist[ip]; ok {
			return true
		}
		return false
	}
	return true
}

func (this *httpListener) auth(w http.ResponseWriter, r *http.Request) bool {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	auth := this.AuthIP(ip)
	if !auth {
		log.Warnln(r.RemoteAddr, r.Method, r.Host)
		this.badAuth(w)
	}
	return auth
}

func (this *httpListener) badAuth(w http.ResponseWriter) {
	http.Error(w, "Unauthorized.", http.StatusUnauthorized)
}
