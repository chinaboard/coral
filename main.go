package main

import (
	"coral/toolkit"
	"fmt"
	"runtime"
	"sync"
)

func main() {
	var c Config
	remoteConfig := toolkit.HttpGet()

	fmt.Printf(`Coral Proxy %s`, version)
	fmt.Println()

	parseConfigString(remoteConfig, &c)

	initSelfListenAddr()
	initLog()
	initAuth()
	initStat()

	initUpstreamPool()

	if config.JudgeByIP {
		initCNIPData()
	}

	if config.Core > 0 {
		runtime.GOMAXPROCS(config.Core)
	}

	go runSSH()

	var wg sync.WaitGroup
	wg.Add(len(listenProxy))
	for _, proxy := range listenProxy {
		go proxy.Serve(&wg)
	}
	wg.Wait()
}
