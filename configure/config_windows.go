package configure

import (
	"os"
	"path"
)

func GetFiletPath(fileName string) string {
	return path.Join(path.Dir(os.Args[0]), fileName+".txt")
}
