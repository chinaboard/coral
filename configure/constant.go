package configure

import "time"

const (
	ConfigFile_coralconfig = "cc"
	ConfigFile_direct      = "direct"
	ConfigFile_proxy       = "proxy"
	ConfigFile_reject      = "reject"
)

type LoadBalanceMode byte

const (
	LoadBalanceBackup LoadBalanceMode = iota
	LoadBalanceHash
	LoadBalanceLatency
)

type ConfigTypeMode byte

const (
	ConfigTypeLocal ConfigTypeMode = iota
	ConfigTypeRemoteFullUrl
	ConfigTypeMonoCloud
)

var DefaultTunnelAllowedPort = []string{
	"22", "80", "443", // ssh, http, https
	"873",                      // rsync
	"143", "220", "585", "993", // imap, imap3, imap4-ssl, imaps
	"109", "110", "473", "995", // pop2, pop3, hybrid-pop, pop3s
	"5222", "5269", // jabber-client, jabber-server
	"5223",                 // jabber-google
	"2401", "3690", "9418", // cvspserver, svn, git
}

var IgnoreOption = map[string]bool{
	"ConfigType":         true,
	"RemoteFullUrl":      true,
	"MonoCloudLoginName": true,
	"MonoCloudPassword":  true,
}

var Option CoralOption

type CoralOption struct {
	LogFile     string          // path for log file
	JudgeByIP   bool            // if false only use DomainType
	DeniedLocal bool            // DeniedLocalAddresses
	LoadBalance LoadBalanceMode // select load balance mode

	TunnelAllowed     bool
	TunnelAllowedPort map[string]bool // allowed ports to create tunnel

	SshServer []string

	// authenticate client
	UserPasswd     string
	UserPasswdFile string // file that contains user:passwd:[port] pairs
	AllowedClient  string
	AuthTimeout    time.Duration

	Core int

	HttpErrorCode int

	// not configurable in config file
	PrintVer bool

	// not config option
	SaveReqLine bool // for http and coral upstream, should save request line from client
	Cert        string
	Key         string
}
