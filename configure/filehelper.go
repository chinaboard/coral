package configure

import (
	"bytes"
	"github.com/chinaboard/coral/bufio"
	"log"
	"os"
)

func getFileContent(fileName string) (lines []string) {
	filePath := GetFiletPath(fileName)
	log.Println("File:", filePath)
	f, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening config file:", err)
		return nil
	}

	ignoreUTF8BOM(f)

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}

	if scanner.Err() != nil {
		log.Fatalf("Error reading cc file: %v\n", scanner.Err())
	}

	f.Close()

	return lines
}

// ignoreUTF8BOM consumes UTF-8 encoded BOM character if present in the file.
func ignoreUTF8BOM(f *os.File) error {
	bom := make([]byte, 3)
	n, err := f.Read(bom)
	if err != nil {
		return err
	}
	if n != 3 {
		return nil
	}
	if bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		log.Println("UTF-8 BOM found")
		return nil
	}
	// No BOM found, seek back
	_, err = f.Seek(-3, 1)
	return err
}

// Return all host IP addresses.
