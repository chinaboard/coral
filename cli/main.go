package main

import (
	"flag"
	"os"

	"github.com/chinaboard/coral/config"
	"github.com/chinaboard/coral/core"

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

	http, err := core.NewHttpListener(conf)
	if err != nil {
		log.Fatalln(err)
		os.Exit(128)
	}

	log.Infof("listen on %s", conf.Common.Address())
	log.Fatalln(http.ListenAndServe())
}
