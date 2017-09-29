package main

import (
	"fmt"
	"runtime"
	"sync"
)

func main() {

	fmt.Println("Coral Proxy ", version)
	fmt.Println()

	initConfig()

	initSelfListenAddr()
	initLog()
	initAuth()
	initStat()

	initUpstreamPool()

	if option.JudgeByIP {
		initCNIPData()
	}

	if option.Core > 0 {
		runtime.GOMAXPROCS(option.Core)
	}

	go runSSH()

	var wg sync.WaitGroup
	wg.Add(len(listenProxy))
	for _, proxy := range listenProxy {
		go proxy.Serve(&wg)
	}
	wg.Wait()

}
