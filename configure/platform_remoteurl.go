package configure

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

func getRemoteFullConfig(url string) (*OptionData, error) {
	var od *OptionData

	if len(url) < 10 || url == "" {
		return nil, errors.New("not found url:" + url)
	}
	remoteData := getUrlString(url)
	err := json.Unmarshal(remoteData, &od)
	return od, err
}

func getRemoteProxyConfig(url string) ([]string, error) {
	var od []string

	if len(url) < 10 || url == "" {
		return nil, errors.New("not found url:" + url)
	}
	remoteData := getUrlString(url)
	err := json.Unmarshal(remoteData, &od)
	return od, err
}

func getUrlString(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	return body
}
