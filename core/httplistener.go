package core

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chinaboard/coral/core/direct"
	"github.com/chinaboard/coral/core/ss"
	"github.com/chinaboard/coral/core/ssr"

	"github.com/chinaboard/coral/core/proxy"

	"github.com/juju/errors"

	"github.com/chinaboard/coral/cache"
	"github.com/chinaboard/coral/config"
	"github.com/chinaboard/coral/leakybuf"
	"github.com/chinaboard/coral/utils"

	log "github.com/sirupsen/logrus"
)

type HttpListener struct {
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

	listener := &HttpListener{
		proxies:   []proxy.Proxy{direct.New(conf.Common.DirectTimeout)},
		cache:     cache.NewCache(time.Minute * 30),
		whitelist: conf.Common.Whitelist,
	}

	listener.srv = &http.Server{
		Addr:    conf.Common.Address(),
		Handler: listener,
	}

	for _, v := range conf.Servers {
		p, err := genProxy(v)
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

func (this *HttpListener) ListenAndServe() error {
	return this.srv.ListenAndServe()
}

func (this *HttpListener) RegisterProxy(proxy proxy.Proxy) (bool, error) {
	if proxy != nil {
		this.Lock()
		defer this.Unlock()
		this.proxies = append(this.proxies, proxy)
		return true, nil
	}
	return false, errors.New("proxy is nil")
}

func (this *HttpListener) RegisterLoadBalance(selectProxyFunc SelectProxyFunc) (bool, error) {
	if selectProxyFunc != nil {
		this.selectProxyFunc = selectProxyFunc
		return true, nil
	}
	return false, errors.New("func errer")
}

func (this *HttpListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			log.Debugf("panic: %v\n", err)
		}
	}()

	if !this.auth(w, r) {
		return
	}

	d, notFound := this.cache.Exist(r.Host)
	if notFound != nil {
		host := strings.Split(r.Host, ":")
		ips, err := net.LookupIP(host[0])
		if err != nil {
			log.Warningln(err, "force use Proxy")
			d = false
		} else {
			ip := ips[0].String()
			d = utils.ShouldDirect(ip)
		}
		this.cache.Set(r.Host, d)
	}

	proxyFunc, err := this.selectProxyFunc(r, this.proxies, d)
	if err != nil {
		log.Errorln(err)
		return
	}
	log.Infoln(proxyFunc.Name(), r.RemoteAddr, r.Method, r.Host)

	if r.Method == "CONNECT" {
		this.HandleConnect(w, r, proxyFunc)
	} else {
		this.HandleHttp(w, r, proxyFunc)
	}

}

func (this *HttpListener) HandleConnect(w http.ResponseWriter, r *http.Request, proxy proxy.Proxy) {
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

	rConn, timeout, errs = proxy.Dial(r.Host)

	if errs != nil {
		log.Errorln(proxy.Name(), "Dial:", err)
		return
	}
	lConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go this.Pipe(lConn, rConn, timeout)
	this.Pipe(rConn, lConn, timeout)
}

func (this *HttpListener) HandleHttp(w http.ResponseWriter, r *http.Request, proxy proxy.Proxy) {
	tr := http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, _, err := proxy.Dial(addr)
			return conn, err
		},
	}

	r.Close = true
	resp, err := tr.RoundTrip(r)
	if err != nil {
		log.Errorln("request error: ", err)
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

func (this *HttpListener) DefaultSelectProxy(r *http.Request, proxies []proxy.Proxy, direct bool) (proxy.Proxy, error) {
	// first direct proxy
	for _, value := range proxies {
		if direct == value.Direct() {
			return value, nil
		}
	}
	return nil, errors.NotFoundf("direct proxy: %s", direct)
}

func (this *HttpListener) Pipe(src, dst net.Conn, timeout time.Duration) error {
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

func (this *HttpListener) auth(w http.ResponseWriter, r *http.Request) bool {
	ip := strings.Split(r.RemoteAddr, ":")[0]
	if len(this.whitelist) > 1 {
		if _, ok := this.whitelist[ip]; ok {
			return true
		}
		log.Warnln(r.RemoteAddr, r.Method, r.Host)
		this.badAuth(w)
		return false
	}
	return true
}

func (this *HttpListener) badAuth(w http.ResponseWriter) {
	http.Error(w, "Unauthorized.", http.StatusUnauthorized)
}

func genProxy(server config.CoralServer) (proxy.Proxy, error) {
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
