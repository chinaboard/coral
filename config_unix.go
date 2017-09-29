// +build darwin freebsd linux netbsd openbsd

package main

import (
	"path"
)

const (
	CcFname     = "cc"
	directFname = "direct"
	proxyFname  = "proxy"
	rejectFname = "reject"

	newLine = "\n"
)

func getDefaultCcFile() string {
	return path.Join(path.Join(getUserHomeDir(), ".coral", CcFname))
}
