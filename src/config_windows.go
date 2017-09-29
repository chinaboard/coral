package main

import (
	"os"
	"path"
)

const (
	CcFname     = "E:\\GoProject\\src\\Coral\\src\\cc.txt"
	directFname = "direct.txt"
	proxyFname  = "proxy.txt"
	rejectFname = "reject.txt"

	newLine = "\r\n"
)

func getDefaultCcFile() string {
	// On windows, put the configuration file in the same directory of coral executable
	// This is not a reliable way to detect binary directory, but it works for double click and run
	dir := path.Join(path.Dir(os.Args[0]), CcFname)
	return dir
}
