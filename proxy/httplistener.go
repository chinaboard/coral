package proxy

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/juju/errors"

	"github.com/chinaboard/coral/cache"
	"github.com/chinaboard/coral/config"
	"github.com/chinaboard/coral/leakybuf"
	"github.com/chinaboard/coral/utils"

	log "github.com/sirupsen/logrus"
)

type HttpListener struct {
	cache   *cache.Cache
	servers []Proxy
	direct  Proxy
}

func NewHttpListener(conf *config.CoralConfig) (*http.Server, error) {
	if len(conf.Servers) == 0 {
		return nil, errors.NotFoundf("server")
	}

	var servers []Proxy
	for n, v := range conf.Servers {
		log.Debugln("parse ..", v.Type, n)
		proxy, err := GenProxy(v)
		if err != nil {
			log.Warningln(err)
			continue
		}
		servers = append(servers, proxy)
	}

	listener := &HttpListener{
		servers: servers,
		direct:  NewDirectProxy(conf.Common.DirectTimeout),
		cache:   cache.NewCache(time.Minute * 30),
	}

	return &http.Server{
		Addr:    conf.Common.Address(),
		Handler: listener,
	}, nil
}

func (this *HttpListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			log.Debugf("panic: %v\n", err)
		}
	}()

	direct, notFound := this.cache.Exist(r.Host)
	if notFound != nil {
		host := strings.Split(r.Host, ":")
		ips, err := net.LookupIP(host[0])
		if err != nil {
			log.Warnf("error looking up Address ip %s, err %s", r.Host, err)
			direct = false
		} else {
			ip := ips[0].String()
			direct = utils.ShouldDirect(ip)
		}
		this.cache.Set(r.Host, direct)
	}

	proxy := this.SelectProxy(direct)

	log.Infoln(proxy.Name(), r.RemoteAddr, r.Method, r.Host)

	if r.Method == "CONNECT" {
		this.HandleConnect(w, r, proxy)
	} else {
		this.HandleHttp(w, r, proxy)
	}

}

func (this *HttpListener) HandleConnect(w http.ResponseWriter, r *http.Request, proxy Proxy) {
	hj, _ := w.(http.Hijacker)
	lConn, _, err := hj.Hijack()
	if err != nil && err != http.ErrHijacked {
		log.Errorln("hijack", err)
		return
	}

	rConn, timeout, err := proxy.Dial(r.Host)
	if err != nil {
		log.Errorln(proxy.Name(), "Dial:", err)
		return
	}
	lConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go this.Pipe(lConn, rConn, timeout)
	this.Pipe(rConn, lConn, timeout)
}

func (this *HttpListener) HandleHttp(w http.ResponseWriter, r *http.Request, proxy Proxy) {
	tr := http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, _, err := proxy.Dial(addr)
			return conn, err
		},
	}

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

func (this *HttpListener) SelectProxy(direct bool) Proxy {
	svr := this.direct
	if direct {
		return svr
	}
	index := rand.Intn(len(this.servers))
	return this.servers[index]
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
