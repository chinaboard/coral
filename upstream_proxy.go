package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
)

// Interface that all types of upstream servers should support.
type UpstreamProxy interface {
	connect(*URL) (net.Conn, error)
	getServer() string // for use in updating server latency
	toString() string  // for upgrading config
}

// Interface for different proxy selection strategy.
type UpstreamPool interface {
	add(UpstreamProxy)
	empty() bool
	// Select a proxy from the pool and connect. May try several servers until
	// one that succees, return nil and error if all upstream servers fail.
	connect(*URL) (net.Conn, error)
}

// Init upstreamProxy to be backup pool. So config parsing have a pool to add
// upstream servers.
var upstreamProxy UpstreamPool = &backupUpstreamPool{}

func initUpstreamPool() {
	backPool, ok := upstreamProxy.(*backupUpstreamPool)
	if !ok {
		panic("initial upstream pool should be backup pool")
	}

	printUpstreamProxy(backPool.upstream)

	if len(backPool.upstream) == 0 {
		info.Println("no upstream proxy server")
		return
	}
	if len(backPool.upstream) == 1 && config.LoadBalance != loadBalanceBackup {
		debug.Println("only 1 upstream, no need for load balance")
		config.LoadBalance = loadBalanceBackup
	}

	switch config.LoadBalance {
	case loadBalanceHash:
		debug.Println("hash upstream pool", len(backPool.upstream))
		upstreamProxy = &hashUpstreamPool{*backPool}
	case loadBalanceLatency:
		debug.Println("latency upstream pool", len(backPool.upstream))
		go updateUpstreamProxyLatency()
		upstreamProxy = newLatencyUpstreamPool(backPool.upstream)
	}
}

func printUpstreamProxy(upstream []UpstreamWithFail) {
	info.Println("avaiable upstream servers:")
	for _, proxyPool := range upstream {
		switch proxy := proxyPool.UpstreamProxy.(type) {
		case *shadowsocksUpstream:
			info.Println("shadowsocks: ", proxy.server)
		case *httpUpstream:
			info.Println("http upstream: ", proxy.server)
		case *socksUpstream:
			info.Println("socks upstream: ", proxy.server)
		case *coralUpstream:
			info.Println("coral upstream: ", proxy.server)
		}
	}
}

type UpstreamWithFail struct {
	UpstreamProxy
	fail int
}

// Backup load balance strategy:
// Select proxy in the order they appear in config.
type backupUpstreamPool struct {
	upstream []UpstreamWithFail
}

func (pp *backupUpstreamPool) empty() bool {
	return len(pp.upstream) == 0
}

func (pp *backupUpstreamPool) add(upstream UpstreamProxy) {
	pp.upstream = append(pp.upstream, UpstreamWithFail{upstream, 0})
}

func (pp *backupUpstreamPool) connect(url *URL) (srvconn net.Conn, err error) {
	return connectInOrder(url, pp.upstream, 0)
}

// Hash load balance strategy:
// Each host will use a proxy based on a hash value.
type hashUpstreamPool struct {
	backupUpstreamPool
}

func (pp *hashUpstreamPool) connect(url *URL) (srvconn net.Conn, err error) {
	start := int(crc32.ChecksumIEEE([]byte(url.Host)) % uint32(len(pp.upstream)))
	debug.Printf("hash host %s try %d upstream first", url.Host, start)
	return connectInOrder(url, pp.upstream, start)
}

func (upstream *UpstreamWithFail) connect(url *URL) (srvconn net.Conn, err error) {
	const maxFailCnt = 30
	srvconn, err = upstream.UpstreamProxy.connect(url)
	if err != nil {
		if upstream.fail < maxFailCnt {
			upstream.fail++
		}
		return
	}
	upstream.fail = 0
	return
}

func connectInOrder(url *URL, pp []UpstreamWithFail, start int) (srvconn net.Conn, err error) {
	const baseFailCnt = 9
	var skipped []int
	nproxy := len(pp)

	if nproxy == 0 {
		return nil, errors.New("no upstream proxy")
	}

	for i := 0; i < nproxy; i++ {
		proxyId := (start + i) % nproxy
		upstream := &pp[proxyId]
		// skip failed server, but try it with some probability
		if upstream.fail > 0 && rand.Intn(upstream.fail+baseFailCnt) != 0 {
			skipped = append(skipped, proxyId)
			continue
		}
		if srvconn, err = upstream.connect(url); err == nil {
			return
		}
	}
	// last resort, try skipped one, not likely to succeed
	for _, skippedId := range skipped {
		if srvconn, err = pp[skippedId].connect(url); err == nil {
			return
		}
	}
	return nil, err
}

type UpstreamWithLatency struct {
	UpstreamProxy
	latency time.Duration
}

type latencyUpstreamPool struct {
	upstream []UpstreamWithLatency
}

func newLatencyUpstreamPool(upstream []UpstreamWithFail) *latencyUpstreamPool {
	lp := &latencyUpstreamPool{}
	for _, p := range upstream {
		lp.add(p.UpstreamProxy)
	}
	return lp
}

func (pp *latencyUpstreamPool) empty() bool {
	return len(pp.upstream) == 0
}

func (pp *latencyUpstreamPool) add(upstream UpstreamProxy) {
	pp.upstream = append(pp.upstream, UpstreamWithLatency{upstream, 0})
}

// Sort interface.
func (pp *latencyUpstreamPool) Len() int {
	return len(pp.upstream)
}

func (pp *latencyUpstreamPool) Swap(i, j int) {
	p := pp.upstream
	p[i], p[j] = p[j], p[i]
}

func (pp *latencyUpstreamPool) Less(i, j int) bool {
	p := pp.upstream
	return p[i].latency < p[j].latency
}

const latencyMax = time.Hour

var latencyMutex sync.RWMutex

func (pp *latencyUpstreamPool) connect(url *URL) (srvconn net.Conn, err error) {
	var lp []UpstreamWithLatency
	// Read slice first.
	latencyMutex.RLock()
	lp = pp.upstream
	latencyMutex.RUnlock()

	var skipped []int
	nproxy := len(lp)
	if nproxy == 0 {
		return nil, errors.New("no upstream proxy")
	}

	for i := 0; i < nproxy; i++ {
		upstream := lp[i]
		if upstream.latency >= latencyMax {
			skipped = append(skipped, i)
			continue
		}
		if srvconn, err = upstream.connect(url); err == nil {
			debug.Println("lowest latency proxy", upstream.getServer())
			return
		}
		upstream.latency = latencyMax
	}
	// last resort, try skipped one, not likely to succeed
	for _, skippedId := range skipped {
		if srvconn, err = lp[skippedId].connect(url); err == nil {
			return
		}
	}
	return nil, err
}

func (upstream *UpstreamWithLatency) updateLatency(wg *sync.WaitGroup) {
	defer wg.Done()
	proxy := upstream.UpstreamProxy
	server := proxy.getServer()

	host, port, err := net.SplitHostPort(server)
	if err != nil {
		panic("split host port upstream server error" + err.Error())
	}

	// Resolve host name first, so latency does not include resolve time.
	ip, err := net.LookupIP(host)
	if err != nil {
		upstream.latency = latencyMax
		return
	}
	ipPort := net.JoinHostPort(ip[0].String(), port)

	const N = 3
	var total time.Duration
	for i := 0; i < N; i++ {
		now := time.Now()
		cn, err := net.Dial("tcp", ipPort)
		if err != nil {
			debug.Println("latency update dial:", err)
			total += time.Minute // 1 minute as penalty
			continue
		}
		total += time.Now().Sub(now)
		cn.Close()

		time.Sleep(5 * time.Millisecond)
	}
	upstream.latency = total / N
	debug.Println("latency", server, upstream.latency)
}

func (pp *latencyUpstreamPool) updateLatency() {
	// Create a copy, update latency for the copy.
	var cp latencyUpstreamPool
	cp.upstream = append(cp.upstream, pp.upstream...)

	// cp.upstream is value instead of pointer, if we use `_, p := range cp.upstream`,
	// the value in cp.upstream will not be updated.
	var wg sync.WaitGroup
	wg.Add(len(cp.upstream))
	for i, _ := range cp.upstream {
		cp.upstream[i].updateLatency(&wg)
	}
	wg.Wait()

	// Sort according to latency.
	sort.Stable(&cp)
	debug.Println("latency lowest proxy", cp.upstream[0].getServer())

	// Update upstream slice.
	latencyMutex.Lock()
	pp.upstream = cp.upstream
	latencyMutex.Unlock()
}

func updateUpstreamProxyLatency() {
	lp, ok := upstreamProxy.(*latencyUpstreamPool)
	if !ok {
		return
	}

	for {
		lp.updateLatency()
		time.Sleep(60 * time.Second)
	}
}

type httpsUpstream struct {
	server     string
	userPasswd string // for upgrade config
	authHeader []byte
}

type httpsConn struct {
	net.Conn
	upstream *httpsUpstream
}

func (s httpsConn) String() string {
	return "https upstream proxy " + s.upstream.server
}

func newHttpsUpstream(server string) *httpsUpstream {
	return &httpsUpstream{server: server}
}

func (hp *httpsUpstream) getServer() string {
	return hp.server
}

func (hp *httpsUpstream) toString() string {
	if hp.userPasswd != "" {
		return fmt.Sprintf("proxy = https://%s@%s", hp.userPasswd, hp.server)
	} else {
		return fmt.Sprintf("proxy = https://%s", hp.server)
	}
}

func (hp *httpsUpstream) initAuth(userPasswd string) {
	if userPasswd == "" {
		return
	}
	hp.userPasswd = userPasswd
	b64 := base64.StdEncoding.EncodeToString([]byte(userPasswd))
	hp.authHeader = []byte(headerProxyAuthorization + ": Basic " + b64 + CRLF)
}

func (hp *httpsUpstream) connect(url *URL) (net.Conn, error) {
	c, err := tls.Dial("tcp", hp.server, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		errl.Printf("can't connect to https upstream %s for %s: %v\n",
			hp.server, url.HostPort, err)
		return nil, err
	}

	debug.Printf("connected to: %s via https upstream: %s\n",
		url.HostPort, hp.server)
	return httpsConn{c, hp}, nil
}

// http upstream proxy
type httpUpstream struct {
	server     string
	userPasswd string // for upgrade config
	authHeader []byte
}

type httpConn struct {
	net.Conn
	upstream *httpUpstream
}

func (s httpConn) String() string {
	return "http upstream proxy " + s.upstream.server
}

func newHttpUpstream(server string) *httpUpstream {
	return &httpUpstream{server: server}
}

func (hp *httpUpstream) getServer() string {
	return hp.server
}

func (hp *httpUpstream) toString() string {
	if hp.userPasswd != "" {
		return fmt.Sprintf("proxy = http://%s@%s", hp.userPasswd, hp.server)
	} else {
		return fmt.Sprintf("proxy = http://%s", hp.server)
	}
}

func (hp *httpUpstream) initAuth(userPasswd string) {
	if userPasswd == "" {
		return
	}
	hp.userPasswd = userPasswd
	b64 := base64.StdEncoding.EncodeToString([]byte(userPasswd))
	hp.authHeader = []byte(headerProxyAuthorization + ": Basic " + b64 + CRLF)
}

func (hp *httpUpstream) connect(url *URL) (net.Conn, error) {
	c, err := net.Dial("tcp", hp.server)
	if err != nil {
		errl.Printf("can't connect to http upstream %s for %s: %v\n",
			hp.server, url.HostPort, err)
		return nil, err
	}
	debug.Printf("connected to: %s via http upstream: %s\n",
		url.HostPort, hp.server)
	return httpConn{c, hp}, nil
}

// shadowsocks upstream proxy
type shadowsocksUpstream struct {
	server string
	method string // method and passwd are for upgrade config
	passwd string
	cipher *ss.Cipher
}

type shadowsocksConn struct {
	net.Conn
	upstream *shadowsocksUpstream
}

func (s shadowsocksConn) String() string {
	return "shadowsocks proxy " + s.upstream.server
}

// In order to use upstream proxy in the order specified in the config file, we
// insert an uninitialized proxy into upstream proxy list, and initialize it
// when all its config have been parsed.

func newShadowsocksUpstream(server string) *shadowsocksUpstream {
	return &shadowsocksUpstream{server: server}
}

func (sp *shadowsocksUpstream) getServer() string {
	return sp.server
}

func (sp *shadowsocksUpstream) toString() string {
	method := sp.method
	if method == "" {
		method = "table"
	}
	return fmt.Sprintf("proxy = ss://%s:%s@%s", method, sp.passwd, sp.server)
}

func (sp *shadowsocksUpstream) initCipher(method, passwd string) {
	sp.method = method
	sp.passwd = passwd
	cipher, err := ss.NewCipher(method, passwd)
	if err != nil {
		Fatal("create shadowsocks cipher:", err)
	}
	sp.cipher = cipher
}

func (sp *shadowsocksUpstream) connect(url *URL) (net.Conn, error) {
	c, err := ss.Dial(url.HostPort, sp.server, sp.cipher.Copy())
	if err != nil {
		errl.Printf("can't connect to shadowsocks upstream %s for %s: %v\n",
			sp.server, url.HostPort, err)
		return nil, err
	}
	debug.Println("connected to:", url.HostPort, "via shadowsocks:", sp.server)
	return shadowsocksConn{c, sp}, nil
}

// coral upstream proxy
type coralUpstream struct {
	server string
	method string
	passwd string
	cipher *ss.Cipher
}

type coralConn struct {
	net.Conn
	upstream *coralUpstream
}

func (s coralConn) String() string {
	return "coral proxy " + s.upstream.server
}

func newCoralUpstream(srv, method, passwd string) *coralUpstream {
	cipher, err := ss.NewCipher(method, passwd)
	if err != nil {
		Fatal("create coral cipher:", err)
	}
	return &coralUpstream{srv, method, passwd, cipher}
}

func (cp *coralUpstream) getServer() string {
	return cp.server
}

func (cp *coralUpstream) toString() string {
	method := cp.method
	if method == "" {
		method = "table"
	}
	return fmt.Sprintf("proxy = coral://%s:%s@%s", method, cp.passwd, cp.server)
}

func (cp *coralUpstream) connect(url *URL) (net.Conn, error) {
	c, err := net.Dial("tcp", cp.server)
	if err != nil {
		errl.Printf("can't connect to coral upstream %s for %s: %v\n",
			cp.server, url.HostPort, err)
		return nil, err
	}
	debug.Printf("connected to: %s via coral upstream: %s\n",
		url.HostPort, cp.server)
	ssconn := ss.NewConn(c, cp.cipher.Copy())
	return coralConn{ssconn, cp}, nil
}

// For socks documentation, refer to rfc 1928 http://www.ietf.org/rfc/rfc1928.txt

var socksError = [...]string{
	1: "General SOCKS server failure",
	2: "Connection not allowed by ruleset",
	3: "Network unreachable",
	4: "Host unreachable",
	5: "Connection refused",
	6: "TTL expired",
	7: "Command not supported",
	8: "Address type not supported",
	9: "to X'FF' unassigned",
}

var socksProtocolErr = errors.New("socks protocol error")

var socksMsgVerMethodSelection = []byte{
	0x5, // version 5
	1,   // n method
	0,   // no authorization required
}

// socks5 upstream proxy
type socksUpstream struct {
	server string
}

type socksConn struct {
	net.Conn
	upstream *socksUpstream
}

func (s socksConn) String() string {
	return "socks proxy " + s.upstream.server
}

func newSocksUpstream(server string) *socksUpstream {
	return &socksUpstream{server}
}

func (sp *socksUpstream) getServer() string {
	return sp.server
}

func (sp *socksUpstream) toString() string {
	return fmt.Sprintf("proxy = socks5://%s", sp.server)
}

func (sp *socksUpstream) connect(url *URL) (net.Conn, error) {
	c, err := net.Dial("tcp", sp.server)
	if err != nil {
		errl.Printf("can't connect to socks upstream %s for %s: %v\n",
			sp.server, url.HostPort, err)
		return nil, err
	}
	hasErr := false
	defer func() {
		if hasErr {
			c.Close()
		}
	}()

	var n int
	if n, err = c.Write(socksMsgVerMethodSelection); n != 3 || err != nil {
		errl.Printf("sending ver/method selection msg %v n = %v\n", err, n)
		hasErr = true
		return nil, err
	}

	// version/method selection
	repBuf := make([]byte, 2)
	_, err = io.ReadFull(c, repBuf)
	if err != nil {
		errl.Printf("read ver/method selection error %v\n", err)
		hasErr = true
		return nil, err
	}
	if repBuf[0] != 5 || repBuf[1] != 0 {
		errl.Printf("socks ver/method selection reply error ver %d method %d",
			repBuf[0], repBuf[1])
		hasErr = true
		return nil, err
	}
	// debug.Println("Socks version selection done")

	// send connect request
	host := url.Host
	port, err := strconv.Atoi(url.Port)
	if err != nil {
		errl.Printf("should not happen, port error %v\n", port)
		hasErr = true
		return nil, err
	}

	hostLen := len(host)
	bufLen := 5 + hostLen + 2 // last 2 is port
	reqBuf := make([]byte, bufLen)
	reqBuf[0] = 5 // version 5
	reqBuf[1] = 1 // cmd: connect
	// reqBuf[2] = 0 // rsv: set to 0 when initializing
	reqBuf[3] = 3 // atyp: domain name
	reqBuf[4] = byte(hostLen)
	copy(reqBuf[5:], host)
	binary.BigEndian.PutUint16(reqBuf[5+hostLen:5+hostLen+2], uint16(port))

	if n, err = c.Write(reqBuf); err != nil || n != bufLen {
		errl.Printf("send socks request err %v n %d\n", err, n)
		hasErr = true
		return nil, err
	}

	// I'm not clear why the buffer is fixed at 10. The rfc document does not say this.
	// Polipo set this to 10 and I also observed the reply is always 10.
	replyBuf := make([]byte, 10)
	if n, err = c.Read(replyBuf); err != nil {
		// Seems that socks server will close connection if it can't find host
		if err != io.EOF {
			errl.Printf("read socks reply err %v n %d\n", err, n)
		}
		hasErr = true
		return nil, errors.New("connection failed (by socks server " + sp.server + "). No such host?")
	}
	// debug.Printf("Socks reply length %d\n", n)

	if replyBuf[0] != 5 {
		errl.Printf("socks reply connect %s VER %d not supported\n", url.HostPort, replyBuf[0])
		hasErr = true
		return nil, socksProtocolErr
	}
	if replyBuf[1] != 0 {
		errl.Printf("socks reply connect %s error %s\n", url.HostPort, socksError[replyBuf[1]])
		hasErr = true
		return nil, socksProtocolErr
	}
	if replyBuf[3] != 1 {
		errl.Printf("socks reply connect %s ATYP %d\n", url.HostPort, replyBuf[3])
		hasErr = true
		return nil, socksProtocolErr
	}

	debug.Println("connected to:", url.HostPort, "via socks server:", sp.server)
	// Now the socket can be used to pass data.
	return socksConn{c, sp}, nil
}
