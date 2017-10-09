package configure

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

var AllConfig OptionData

type OptionData struct {
	Content      []string
	DirectDomain []string
	ProxyDomain  []string
	RejectDomain []string
}

func (od *OptionData) getLocalConfig() error {

	od.Content = getFileContent(ConfigFile_coralconfig)
	od.DirectDomain = getFileContent(ConfigFile_direct)
	od.ProxyDomain = getFileContent(ConfigFile_proxy)
	od.RejectDomain = getFileContent(ConfigFile_reject)

	return nil
}

func (od *OptionData) getValue(key, defaultValue string) string {

	for _, value := range od.Content {
		value = strings.TrimSpace(value)
		if strings.Index(value, key) == 0 {
			v := strings.SplitN(value, "=", 2)
			if len(v) == 2 {
				return strings.TrimSpace(v[1])
			}
		}
	}
	return defaultValue
}

func InitOption() {

	AllConfig.getLocalConfig()

	val := strings.TrimSpace(AllConfig.getValue("ConfigType", "local"))

	parser := reflect.ValueOf(optionParser{})
	zeroMethod := reflect.Value{}

	methodName := "Parse" + val
	method := parser.MethodByName(methodName)
	if method == zeroMethod {
		log.Printf("no such ConfigType \"%s\"\n", val)
	}
	log.Printf("configType : %s", val)

	method.Call(nil)
}

type optionParser struct{}

func (op optionParser) ParseLocal() {
}

func (op optionParser) ParseRemoteProxy() {
	url := AllConfig.getValue("RemoteUrl", "")
	cf, err := getRemoteProxyConfig(url)
	if err == nil {
		AllConfig.Content = cf
	}
}

func (op optionParser) ParseRemoteFull() {
	url := AllConfig.getValue("RemoteUrl", "")
	cf, err := getRemoteFullConfig(url)
	if err == nil {
		AllConfig.Content = cf.Content
		AllConfig.ProxyDomain = cf.ProxyDomain
		AllConfig.RejectDomain = cf.RejectDomain
		AllConfig.DirectDomain = cf.DirectDomain
	}
}

func (op optionParser) ParseMonoCloudAll() {

	UserInfo.Username = AllConfig.getValue("MonoCloudLoginName", "")
	UserInfo.Password = AllConfig.getValue("MonoCloudPassword", "")
	monoServers, err := getMonoCloudServers()
	if err != nil {
		log.Println("Error get MonoCloud config:", err)
	}

	for k, v := range AllConfig.Content {
		v = strings.TrimSpace(v)
		if strings.Index(v, "proxy") == 0 {
			AllConfig.Content[k] = fmt.Sprintf("#%s", v)
		}
	}
	//monoProxy
	for _, value := range monoServers.SS {
		AllConfig.Content = append(AllConfig.Content, value)
	}
	for _, value := range monoServers.Tls {
		AllConfig.Content = append(AllConfig.Content, value)
	}

	log.Println("Init Monocloud servers:", len(monoServers.SS)+len(monoServers.Tls))

}

func (op optionParser) ParseMonoCloudSS() {
	UserInfo.Username = AllConfig.getValue("MonoCloudLoginName", "")
	UserInfo.Password = AllConfig.getValue("MonoCloudPassword", "")
	monoServers, err := getMonoCloudServers()
	if err != nil {
		log.Println("Error get MonoCloud config:", err)
	}

	for k, v := range AllConfig.Content {
		v = strings.TrimSpace(v)
		if strings.Index(v, "proxy") == 0 {
			AllConfig.Content[k] = fmt.Sprintf("#%s", v)
		}
	}
	//monoProxy
	for _, value := range monoServers.SS {
		AllConfig.Content = append(AllConfig.Content, value)
	}

	log.Println("Init Monocloud servers:", len(monoServers.SS))

}

func (op optionParser) ParseMonoCloudTls() {
	UserInfo.Username = AllConfig.getValue("MonoCloudLoginName", "")
	UserInfo.Password = AllConfig.getValue("MonoCloudPassword", "")
	monoServers, err := getMonoCloudServers()
	if err != nil {
		log.Println("Error get MonoCloud config:", err)
	}

	for k, v := range AllConfig.Content {
		v = strings.TrimSpace(v)
		if strings.Index(v, "proxy") == 0 {
			AllConfig.Content[k] = fmt.Sprintf("#%s", v)
		}
	}
	//monoProxy
	for _, value := range monoServers.Tls {
		AllConfig.Content = append(AllConfig.Content, value)
	}

	log.Println("Init Monocloud servers:", len(monoServers.Tls))

}
