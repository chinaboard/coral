package toolkit

import (
	"io/ioutil"
	"net/http"
	"strings"
)

func HttpGet() []string {
	resp, err := http.Get("http://localhost:12345/hello")
	if err != nil {
		// handle error
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}

	return strings.Split(string(body), "\r\n")
}
