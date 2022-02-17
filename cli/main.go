package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/chinaboard/coral/config"
	"github.com/chinaboard/coral/core"

	_ "net/http/pprof"

	_ "github.com/chinaboard/coral/utils/version"
	log "github.com/sirupsen/logrus"
)

func main() {
	configFile := ""
	flag.StringVar(&configFile, "config", "", "Configuration filename")
	flag.Parse()

	conf, err := config.ParseFileConfig(configFile)
	if err != nil {
		log.Fatalln(err)
		return
	}

	httpls, err := core.NewHttpListener(conf)
	if err != nil {
		log.Fatalln(err)
		os.Exit(128)
	}

	log.Infof("listen on %s", conf.Common.Address())
	go http.ListenAndServe(fmt.Sprint("0.0.0.0:", conf.Common.Port+1), nil)
	log.Fatalln(httpls.ListenAndServe())
}
