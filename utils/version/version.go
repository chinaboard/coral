package version

import (
	"fmt"
	"os"

	"github.com/chinaboard/coral/utils/data"
)

var (
	BuildVersion = "1.0.0.20200917"
)

func init() {
	args := os.Args
	if nil == args || len(args) < 2 {
		return
	}
	if "-v" == args[1] {
		fmt.Println("Coral: ", BuildVersion, "CNIPDataNum", len(data.CNIPDataNum), "CNIPDataStart", len(data.CNIPDataStart))
	}
	os.Exit(0)
}
