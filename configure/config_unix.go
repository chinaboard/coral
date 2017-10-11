// +build darwin freebsd linux netbsd openbsd

package configure

import (
	"fmt"
	"os"
	"path"
)

func GetFiletPath(fileName string) string {
	return path.Join(path.Join(getUserHomeDir(), ".coral", fileName))
}

func getUserHomeDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		fmt.Println("HOME environment variable is empty")
	}
	return home
}
