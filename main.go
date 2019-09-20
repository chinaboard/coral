package main

import (
	"os"

	"github.com/chinaboard/coral/config"
	"github.com/chinaboard/coral/proxy"

	log "github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.ParseFileConfig("")
	if err != nil {
		log.Fatalln(err)
		return
	}

	listener, err := proxy.NewHttpListener(conf)
	if err != nil {
		log.Fatalln(err)
		os.Exit(128)
	}

	log.Infof("listen on %s", conf.Common.Address())
	log.Fatalln(listener.ListenAndServe())
}
