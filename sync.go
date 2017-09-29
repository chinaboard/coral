package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type conifgData struct {
	config       []string
	directDomain []string
	proxyDomain  []string
	rejectDomain []string
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
