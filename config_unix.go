// +build darwin freebsd linux netbsd openbsd

package main

import (
	"path"
)

const (
	ccFname     = "cc"
	directFname = "direct"
	proxyFname  = "proxy"
	rejectFname = "reject"

	newLine = "\n"
)

func getRcFile() string {
	return path.Join(path.Join(getUserHomeDir(), ".coral", rcFname))
}

func getDirectFile() string {
	return path.Join(path.Join(getUserHomeDir(), ".coral", directFname))
}

func getProxyFile() string {
	return path.Join(path.Join(getUserHomeDir(), ".coral", proxyFname))
}

func getRejectFile() string {
	return path.Join(path.Join(getUserHomeDir(), ".coral", rejectFname))
}
