package configure

import (
	"encoding/json"
	"golang.org/x/oauth2"
	"io/ioutil"
	"os"
)

type TokenCache interface {
	// Reads a cached token. It may return nil if none is cached.
	Read() (*oauth2.Token, error)
	// Write writes a token to the cache.
	Write(*oauth2.Token,string) error
}

type FileCache struct {
	filename string
}

func NewFileCache() (cache *FileCache) {
	return &FileCache{filename: "token.cache"}
}

func (f *FileCache) Write(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err == nil {
		err = ioutil.WriteFile(f.filename, data, 0644)
	}
	return err
}

func (f *FileCache) Read() (token *oauth2.Token, err error) {
	data, err := ioutil.ReadFile(f.filename)
	if os.IsNotExist(err) {
		// no token has cached before, skip reading
		return nil, nil
	}
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &token); err != nil {
		return
	}
	return
}
