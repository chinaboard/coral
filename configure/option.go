package configure

import (
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

	val := strings.TrimSpace(AllConfig.getValue("ConfigType", "Local"))

	parser := reflect.ValueOf(optionParser{})
	zeroMethod := reflect.Value{}

	methodName := "Parse" + val
	method := parser.MethodByName(methodName)
	if method == zeroMethod {
		log.Fatalf("no such ConfigType \"%s\"\n", val)
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
	} else {
		log.Println("get remote proxy err ", err)
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
	} else {
		log.Println("get remote proxy err ", err)
	}
}
