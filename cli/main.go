package main

import (
	"flag"
	"os"

	"github.com/chinaboard/coral/backend"
	"github.com/chinaboard/coral/config"

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

	listener, err := backend.NewHttpListener(conf)
	if err != nil {
		log.Fatalln(err)
		os.Exit(128)
	}

	log.Infof("listen on %s", conf.Common.Address())
	log.Fatalln(listener.ListenAndServe())
}
