package config

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/user"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/vaughan0/go-ini"

	log "github.com/sirupsen/logrus"
)

type CoralConfig struct {
	Common  CoralConfigCommon
	Servers map[string]CoralServer
}

type CoralServer struct {
	Name          string        `json:"name"`
	Type          string        `json:"type"`
	Host          string        `json:"host"`
	Port          string        `json:"port"`
	Method        string        `json:"method"`
	Password      string        `json:"password"`
	Obfs          string        `json:"obfs"`
	ObfsParam     string        `json:"obfsParam"`
	Protocol      string        `json:"protocol"`
	ProtocolParam string        `json:"protocolParam"`
	ReadTimeout   time.Duration `json:"readTimeout"`
}

func (c CoralServer) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

type CoralConfigCommon struct {
	Host          string        `json:"address"`
	Port          int           `json:"port"`
	DirectTimeout time.Duration `json:"directTimeout"`
}

func (c CoralConfigCommon) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC3339})
}

func ParseFileConfig(configFile string) (*CoralConfig, error) {
	if configFile == "" {
		usr, err := user.Current()
		if err == nil {
			configFile = usr.HomeDir + "/.coral/cc.ini"
			log.Infoln(configFile)
		}
	}

	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	return ParseIniConfig(string(buf))
}

func ParseRemoteConfig(url string) (*CoralConfig, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("Unexpected statusCode " + resp.Status)
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return ParseIniConfig(string(bytes))
}

func ParseIniConfig(str string) (*CoralConfig, error) {
	reader := strings.NewReader(str)

	conf, err := ini.Load(reader)
	if err != nil {
		return nil, errors.Errorf("parse ini conf file error: %v", err)
	}

	cfg := GetDefaultConfig()
	var (
		tmpStr string
		ok     bool
		v      int
	)
	if tmpStr, ok = conf.Get("common", "host"); ok {
		cfg.Common.Host = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "port"); ok {
		v, err = strconv.Atoi(tmpStr)
		if err != nil {
			err = errors.Errorf("Parse conf error: invalid port")
			return nil, err
		}
		cfg.Common.Port = v
	}

	for name, section := range conf {
		if name == "common" {
			continue
		}
		if value, err := UnmarshalServerFormSection(name, section); err != nil {
			return nil, err
		} else {
			cfg.Servers[name] = value
		}
	}

	return &cfg, nil
}

func UnmarshalServerFormSection(name string, section ini.Section) (CoralServer, error) {
	cfg := CoralServer{Name: name}
	var (
		tmpStr string
		ok     bool
	)

	//default readTimeout
	cfg.ReadTimeout = time.Second * 10
	if tmpStr, ok = section["readTimeout"]; ok {
		if v, err := strconv.Atoi(tmpStr); err != nil {
			return cfg, errors.New("Parse conf error: invalid readTimeout")
		} else {
			cfg.ReadTimeout = time.Second * time.Duration(v)
		}
	}
	if tmpStr, ok = section["type"]; ok {
		cfg.Type = tmpStr
	} else {
		return cfg, errors.NotFoundf("type")
	}

	//todo use reflect
	ss := []string{"Host", "Port", "Method", "Password"}
	ssr := []string{"Obfs", "ObfsParam", "Protocol", "ProtocolParam"}

	unmarshal := func(keyList []string, ccs *CoralServer) error {
		value := reflect.ValueOf(ccs).Elem()
		for _, name := range keyList {
			key := strings.ToLower(string(name[0])) + name[1:]
			if tmpStr, ok = section[key]; ok {
				value.FieldByName(name).Set(reflect.ValueOf(tmpStr))
			} else {
				return errors.NotFoundf("Parse conf error: %s", key)
			}
		}
		return nil
	}

	cfg.Type = strings.ToLower(cfg.Type)

	switch cfg.Type {
	case "ssr":
		if err := unmarshal(append(ss, ssr...), &cfg); err != nil {
			return cfg, err
		}
	case "ss":
		if err := unmarshal(ss, &cfg); err != nil {
			return cfg, err
		}
	default:
		return cfg, errors.NotSupportedf(cfg.Type)
	}
	return cfg, nil
}

func GetDefaultConfig() CoralConfig {
	return CoralConfig{
		Common: CoralConfigCommon{
			Host:          "127.0.0.1",
			Port:          5438,
			DirectTimeout: time.Second * 10,
		},
		Servers: map[string]CoralServer{},
	}
}
