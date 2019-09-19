package main

import (
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
	listener := proxy.NewHttpListener(conf)
	log.Infof("listen on %s", conf.Common.Address())
	log.Fatalln(listener.ListenAndServe())
}
