package main

import (
	"os"
	"path"
)

const (
	ccFname     = "cc.txt"
	directFname = "direct.txt"
	proxyFname  = "proxy.txt"
	rejectFname = "reject.txt"

	newLine = "\r\n"
)

func getCcFile() string {
	return path.Join(path.Dir(os.Args[0]), ccFname)
}

func getDirectFile() string {
	return path.Join(path.Dir(os.Args[0]), directFname)
}

func getProxyFile() string {
	return path.Join(path.Dir(os.Args[0]), proxyFname)
}

func getRejectFile() string {
	return path.Join(path.Dir(os.Args[0]), rejectFname)
}
