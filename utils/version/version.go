package version

import (
	"fmt"
	"os"
)

var (
	BuildVersion  = "1.0.0.20200408"
)

func init() {
	args := os.Args
	if nil == args || len(args) < 2 {
		return
	}
	if "-v" == args[1] {
		fmt.Printf("Coral: v%s \n", BuildVersion)
	}
	os.Exit(0)
}