package main

import (
	"errors"
	"github.com/chinaboard/coral/configure"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	version           = "1.2.0"
	defaultListenAddr = "127.0.0.1:5438"
)

// allow the same tunnel ports as polipo

var option = configure.Option

func parseBool(v, msg string) bool {
	switch v {
	case "true":
		return true
	case "false":
		return false
	case "0":
		return false
	default:
		Fatalf("%s should be true or false\n", msg)
	}
	return false
}

func parseInt(val, msg string) (i int) {
	var err error
	if i, err = strconv.Atoi(val); err != nil {
		Fatalf("%s should be an integer\n", msg)
	}
	return
}

func parseDuration(val, msg string) (d time.Duration) {
	var err error
	if d, err = time.ParseDuration(val); err != nil {
		Fatalf("%s %v\n", msg, err)
	}
	return
}

func checkServerAddr(addr string) error {
	_, _, err := net.SplitHostPort(addr)
	return err
}

func isUserPasswdValid(val string) bool {
	arr := strings.SplitN(val, ":", 2)
	if len(arr) != 2 || arr[0] == "" || arr[1] == "" {
		return false
	}
	return true
}

// proxyParser provides functions to parse different types of upstream proxy
type proxyParser struct{}

func (pp proxyParser) ProxySocks5(val string) {
	if err := checkServerAddr(val); err != nil {
		Fatal("upstream socks server", err)
	}
	upstreamProxy.add(newSocksUpstream(val))
}

func (pp proxyParser) ProxyHttp(val string) {
	var userPasswd, server string

	idx := strings.LastIndex(val, "@")
	if idx == -1 {
		server = val
	} else {
		userPasswd = val[:idx]
		server = val[idx+1:]
	}

	if err := checkServerAddr(server); err != nil {
		Fatal("upstream http server", err)
	}

	option.SaveReqLine = true

	upstream := newHttpUpstream(server)
	upstream.initAuth(userPasswd)
	upstreamProxy.add(upstream)
}

func (pp proxyParser) ProxyHttps(val string) {
	var userPasswd, server string

	idx := strings.LastIndex(val, "@")
	if idx == -1 {
		server = val
	} else {
		userPasswd = val[:idx]
		server = val[idx+1:]
	}

	if err := checkServerAddr(server); err != nil {
		Fatal("upstream http server", err)
	}

	option.SaveReqLine = true

	upstream := newHttpsUpstream(server)
	upstream.initAuth(userPasswd)
	upstreamProxy.add(upstream)
}

// Parse method:passwd@server:port
func parseMethodPasswdServer(val string) (method, passwd, server string, err error) {
	// Use the right-most @ symbol to seperate method:passwd and server:port.
	idx := strings.LastIndex(val, "@")
	if idx == -1 {
		err = errors.New("requires both encrypt method and password")
		return
	}

	methodPasswd := val[:idx]
	server = val[idx+1:]
	if err = checkServerAddr(server); err != nil {
		return
	}

	// Password can have : inside, but I don't recommend this.
	arr := strings.SplitN(methodPasswd, ":", 2)
	if len(arr) != 2 {
		err = errors.New("method and password should be separated by :")
		return
	}
	method = arr[0]
	passwd = arr[1]
	return
}

// parse shadowsocks proxy
func (pp proxyParser) ProxySs(val string) {
	method, passwd, server, err := parseMethodPasswdServer(val)
	if err != nil {
		Fatal("shadowsocks upstream", err)
	}
	upstream := newShadowsocksUpstream(server)
	upstream.initCipher(method, passwd)
	upstreamProxy.add(upstream)
}

func (pp proxyParser) ProxyCoral(val string) {
	method, passwd, server, err := parseMethodPasswdServer(val)
	if err != nil {
		Fatal("coral upstream", err)
	}

	if err := checkServerAddr(server); err != nil {
		Fatal("upstream coral server", err)
	}

	option.SaveReqLine = true
	upstream := newCoralUpstream(server, method, passwd)
	upstreamProxy.add(upstream)
}

// listenParser provides functions to parse different types of listen addresses
type listenParser struct{}

func (lp listenParser) ListenHttp(val string, proto string) {
	arr := strings.Fields(val)
	if len(arr) > 2 {
		Fatal("too many fields in listen =", proto, val)
	}

	var addr, addrInPAC string
	addr = arr[0]
	if len(arr) == 2 {
		addrInPAC = arr[1]
	}

	if err := checkServerAddr(addr); err != nil {
		Fatal("listen", proto, "server", err)
	}
	addListenProxy(newHttpProxy(addr, addrInPAC, proto))
}

func (lp listenParser) ListenCoral(val string) {
	method, passwd, addr, err := parseMethodPasswdServer(val)
	if err != nil {
		Fatal("listen coral", err)
	}
	addListenProxy(newCoralProxy(method, passwd, addr))
}

// configParser provides functions to parse options in option file.
type configParser struct{}

func (p configParser) ParseProxy(val string) {
	parser := reflect.ValueOf(proxyParser{})
	zeroMethod := reflect.Value{}

	arr := strings.Split(val, "://")
	if len(arr) != 2 {
		Fatal("proxy has no protocol specified:", val)
	}
	protocol := arr[0]

	methodName := "Proxy" + strings.ToUpper(protocol[0:1]) + protocol[1:]
	method := parser.MethodByName(methodName)
	if method == zeroMethod {
		Fatalf("no such protocol \"%s\"\n", arr[0])
	}
	args := []reflect.Value{reflect.ValueOf(arr[1])}
	method.Call(args)
}

func (p configParser) ParseListen(val string) {
	parser := reflect.ValueOf(listenParser{})
	zeroMethod := reflect.Value{}

	var protocol, server string
	arr := strings.Split(val, "://")
	if len(arr) == 1 {
		protocol = "http"
		server = val
	} else {
		protocol = arr[0]
		server = arr[1]
	}

	methodName := "Listen" + strings.ToUpper(protocol[0:1]) + protocol[1:]
	if methodName == "ListenHttps" {
		methodName = "ListenHttp"
	}
	method := parser.MethodByName(methodName)
	if method == zeroMethod {
		Fatalf("no such listen protocol \"%s\"\n", arr[0])
	}
	if methodName == "ListenCoral" {
		method.Call([]reflect.Value{reflect.ValueOf(server)})
	} else {
		method.Call([]reflect.Value{reflect.ValueOf(server), reflect.ValueOf(protocol)})
	}
}

func (p configParser) ParseLogFile(val string) {
	option.LogFile = expandTilde(val)
}

func (p configParser) ParseAddrInPAC(val string) {
	arr := strings.Split(val, ",")
	for i, s := range arr {
		if s == "" {
			continue
		}
		s = strings.TrimSpace(s)
		host, _, err := net.SplitHostPort(s)
		if err != nil {
			Fatal("proxy address in PAC", err)
		}
		if host == "0.0.0.0" {
			Fatal("can't use 0.0.0.0 as proxy address in PAC")
		}
		if hp, ok := listenProxy[i].(*httpProxy); ok {
			hp.addrInPAC = s
		} else {
			Fatal("can't specify address in PAC for non http proxy")
		}
	}
}

func (p configParser) ParseTunnelAllowedPort(val string) {
	arr := strings.Split(val, ",")
	for _, s := range arr {
		s = strings.TrimSpace(s)
		if _, err := strconv.Atoi(s); err != nil {
			Fatal("tunnel allowed ports", err)
		}
		option.TunnelAllowedPort[s] = true
	}
}

func (p configParser) ParseSocksUpstream(val string) {
	var pp proxyParser
	pp.ProxySocks5(val)
}

func (p configParser) ParseSshServer(val string) {
	arr := strings.Split(val, ":")
	if len(arr) == 2 {
		val += ":22"
	} else if len(arr) == 3 {
		if arr[2] == "" {
			val += "22"
		}
	} else {
		Fatal("sshServer should be in the form of: user@server:local_socks_port[:server_ssh_port]")
	}
	// add created socks server
	p.ParseSocksUpstream("127.0.0.1:" + arr[1])
	option.SshServer = append(option.SshServer, val)
}

var httpProtocol struct {
	upstream  *httpUpstream
	serverCnt int
	passwdCnt int
}

func (p configParser) ParseHttpUpstream(val string) {
	if err := checkServerAddr(val); err != nil {
		Fatal("upstream http server", err)
	}
	option.SaveReqLine = true
	httpProtocol.upstream = newHttpUpstream(val)
	upstreamProxy.add(httpProtocol.upstream)
	httpProtocol.serverCnt++
}

func (p configParser) ParseHttpUserPasswd(val string) {
	if !isUserPasswdValid(val) {
		Fatal("httpUserPassword syntax wrong, should be in the form of user:passwd")
	}
	if httpProtocol.passwdCnt >= httpProtocol.serverCnt {
		Fatal("must specify httpUpstream before corresponding httpUserPasswd")
	}
	httpProtocol.upstream.initAuth(val)
	httpProtocol.passwdCnt++
}

func (p configParser) ParseLoadBalance(val string) {
	switch val {
	case "backup":
		option.LoadBalance = configure.LoadBalanceBackup
	case "hash":
		option.LoadBalance = configure.LoadBalanceHash
	case "latency":
		option.LoadBalance = configure.LoadBalanceLatency
	default:
		Fatalf("invalid loadBalance mode: %s\n", val)
	}
}

var shadowProtocol struct {
	upstream *shadowsocksUpstream
	passwd   string
	method   string

	serverCnt int
	passwdCnt int
	methodCnt int
}

func (p configParser) ParseShadowSocks(val string) {
	if shadowProtocol.serverCnt-shadowProtocol.passwdCnt > 1 {
		Fatal("must specify shadowPasswd for every shadowSocks server")
	}
	// create new shadowsocks upstream if both server and password are given
	// previously
	if shadowProtocol.upstream != nil && shadowProtocol.serverCnt == shadowProtocol.passwdCnt {
		if shadowProtocol.methodCnt < shadowProtocol.serverCnt {
			shadowProtocol.method = ""
			shadowProtocol.methodCnt = shadowProtocol.serverCnt
		}
		shadowProtocol.upstream.initCipher(shadowProtocol.method, shadowProtocol.passwd)
	}
	if val == "" { // the final call
		shadowProtocol.upstream = nil
		return
	}
	if err := checkServerAddr(val); err != nil {
		Fatal("shadowsocks server", err)
	}
	shadowProtocol.upstream = newShadowsocksUpstream(val)
	upstreamProxy.add(shadowProtocol.upstream)
	shadowProtocol.serverCnt++
}

func (p configParser) ParseShadowPasswd(val string) {
	if shadowProtocol.passwdCnt >= shadowProtocol.serverCnt {
		Fatal("must specify shadowSocks before corresponding shadowPasswd")
	}
	if shadowProtocol.passwdCnt+1 != shadowProtocol.serverCnt {
		Fatal("must specify shadowPasswd for every shadowSocks")
	}
	shadowProtocol.passwd = val
	shadowProtocol.passwdCnt++
}

func (p configParser) ParseShadowMethod(val string) {
	if shadowProtocol.methodCnt >= shadowProtocol.serverCnt {
		Fatal("must specify shadowSocks before corresponding shadowMethod")
	}
	// shadowMethod is optional
	shadowProtocol.method = val
	shadowProtocol.methodCnt++
}

func checkShadowsocks() {
	if shadowProtocol.serverCnt != shadowProtocol.passwdCnt {
		Fatal("number of shadowsocks server and password does not match")
	}
	// parse the last shadowSocks option again to initialize the last
	// shadowsocks server
	parser := configParser{}
	parser.ParseShadowSocks("")
}

// Put actual authentication related option parsing in auth.go, so option.go
// doesn't need to know the details of authentication implementation.

func (p configParser) ParseUserPasswd(val string) {
	option.UserPasswd = val
	if !isUserPasswdValid(option.UserPasswd) {
		Fatal("userPassword syntax wrong, should be in the form of user:passwd")
	}
}

func (p configParser) ParseUserPasswdFile(val string) {
	err := isFileExists(val)
	if err != nil {
		Fatal("userPasswdFile:", err)
	}
	option.UserPasswdFile = val
}

func (p configParser) ParseAllowedClient(val string) {
	option.AllowedClient = val
}

func (p configParser) ParseAuthTimeout(val string) {
	option.AuthTimeout = parseDuration(val, "authTimeout")
}

func (p configParser) ParseCore(val string) {
	option.Core = parseInt(val, "core")
}

func (p configParser) ParseHttpErrorCode(val string) {
	option.HttpErrorCode = parseInt(val, "httpErrorCode")
}

func (p configParser) ParseJudgeByIP(val string) {
	option.JudgeByIP = parseBool(val, "judgeByIP")
}

func (p configParser) ParseDeniedLocal(val string) {
	option.DeniedLocal = parseBool(val, "DeniedLocal")
}

func (p configParser) ParseTunnelAllowed(val string) {
	option.TunnelAllowed = parseBool(val, "TunnelAllowed")
}

func (p configParser) ParseCert(val string) {
	option.Cert = val
}

func (p configParser) ParseKey(val string) {
	option.Key = val
}

func initConfig() {

	option.JudgeByIP = true
	option.DeniedLocal = true
	option.TunnelAllowed = false
	option.AuthTimeout = 2 * time.Hour

	configure.InitOption()

	config := configure.AllConfig

	initLinesConfig(config.Content)

	initDomainList(config.DirectDomain, domainTypeDirect)
	initDomainList(config.ProxyDomain, domainTypeProxy)
	initDomainList(config.RejectDomain, domainTypeReject)

	checkConfig()
}

func initLinesConfig(lines []string) {
	parser := reflect.ValueOf(configParser{})
	zeroMethod := reflect.Value{}
	for index, line := range lines {

		if line == "" || line[0] == '#' {
			continue
		}

		v := strings.SplitN(line, "=", 2)
		if len(v) != 2 {
			Fatal("option syntax error on line", index+1)
		}
		key, val := strings.TrimSpace(v[0]), strings.TrimSpace(v[1])

		_, found := configure.IgnoreOption[key]
		if found {
			continue
		}

		methodName := "Parse" + strings.ToUpper(key[0:1]) + key[1:]
		method := parser.MethodByName(methodName)
		if method == zeroMethod {
			errl.Printf("no such option \"%s\"\n", key)
			continue
		}
		// for backward compatibility, allow empty string in shadowMethod and logFile
		if val == "" && key != "shadowMethod" && key != "logFile" {
			Fatalf("empty %s, please comment or remove unused option\n", key)
		}
		args := []reflect.Value{reflect.ValueOf(val)}
		method.Call(args)
	}
}

// Must call checkConfig before using option.
func checkConfig() {
	checkShadowsocks()
	// listenAddr must be handled first, as addrInPAC dependends on this.
	if listenProxy == nil {
		listenProxy = []Proxy{newHttpProxy(defaultListenAddr, "", "http")}
	}
}
