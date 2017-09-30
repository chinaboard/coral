package main

import (
	"coral/bufio"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
)

type conifgData struct {
	Config       []string
	DirectDomain []string
	ProxyDomain  []string
	RejectDomain []string
}

var configData conifgData

func syncConfigData() *conifgData {

	remoteData := get("http://localhost:12345/config")

	json.Unmarshal(remoteData, &configData)

	return &configData
}

func get(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		errl.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errl.Println(err)
	}
	return body
}

func initLocalConfig() (*conifgData, error) {

	if err := isFileExists(getCcFile()); err != nil {
		return nil, err
	}
	configData.Config = getLocalFile(getCcFile())
	configData.DirectDomain = getLocalFile(getDirectFile())
	configData.ProxyDomain = getLocalFile(getProxyFile())
	configData.RejectDomain = getLocalFile(getRejectFile())
	return &configData, nil
}

func getLocalFile(filePath string) (lines []string) {
	debug.Println("File:", filePath)
	f, err := os.Open(filePath)
	if err != nil {
		Fatal("Error opening config file:", err)
	}

	IgnoreUTF8BOM(f)

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if scanner.Err() != nil {
		Fatalf("Error reading cc file: %v\n", scanner.Err())
	}

	f.Close()

	return lines
}
