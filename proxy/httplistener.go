package proxy

import (
	"context"
	"coral/cache"
	"coral/config"
	"coral/utils"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	log "github.com/sirupsen/logrus"
)

type HttpListener struct {
	cache   *cache.Cache
	servers []Proxy
	direct  Proxy
}

func NewHttpListener(conf *config.CoralConfig) *http.Server {

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
		//todo directTimeout config
		direct: NewDirectProxy(time.Second * 30),
		cache:  cache.NewCache(time.Minute * 30),
	}

	return &http.Server{
		Addr:    conf.Common.Address(),
		Handler: listener,
	}
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
			this.cache.Set(r.Host, direct)
		}
	}

	dial, name := this.chooseDial(direct)

	log.Infoln(name, r.RemoteAddr, r.Method, r.Host)

	if r.Method == "CONNECT" {
		this.HandleConnect(w, r, dial)
	} else {
		this.HandleHttp(w, r, dial)
	}

}

func (this *HttpListener) HandleConnect(w http.ResponseWriter, r *http.Request, dial DialFunc) {
	hj, _ := w.(http.Hijacker)
	lConn, _, err := hj.Hijack()
	if err != nil && err != http.ErrHijacked {
		log.Errorln("hijack", err)
		return
	}

	rConn, err := dial(r.Host)
	if err != nil {
		log.Errorln("dial:", err)
		return
	}

	lConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go ss.PipeThenClose(lConn, rConn, nil)
	ss.PipeThenClose(rConn, lConn, nil)
}

func (this *HttpListener) HandleHttp(w http.ResponseWriter, r *http.Request, dial DialFunc) {
	tr := http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dial(addr)
		},
	}

	resp, err := tr.RoundTrip(r)
	if err != nil {
		log.Error("request error: ", err)
		return
	}
	defer resp.Body.Close()

	// copy headers
	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// copy body
	io.Copy(w, resp.Body)
}

func (this *HttpListener) chooseDial(direct bool) (DialFunc, string) {
	svr := this.direct
	if direct {
		return svr.Dial, svr.Name()
	}
	index := rand.Intn(len(this.servers))
	svr = this.servers[index]
	return svr.Dial, svr.Name()
}
