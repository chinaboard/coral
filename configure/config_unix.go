// +build darwin freebsd linux netbsd openbsd

package configure

import (
	"path"
)

func GetFiletPath(fileName string) string {
	return path.Join(path.Join(getUserHomeDir(), ".coral", fileName))
}
