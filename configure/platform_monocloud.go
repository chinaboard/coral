package configure

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	baseUrl string = "https://monocloud.net/"
	apiUrl  string = baseUrl + "api/"
)

var MonoCloudEndpoint = oauth2.Endpoint{TokenURL: baseUrl + "oauth/token"}

var UserInfo struct {
	Username string
	Password string
}

type MonoCloudServer interface {
	ToString() string
}

type DarkCloudServer struct {
	Encryption string `json:"encryption"`
	Protocol   string `json:"protocol"`
	Password   string `json:"password"`
	HostName   string `json:"hostname"`
	Port       int    `json:"port"`
	Enable     int    `json:"enable"`
	SSR        int    `json:"ssr"`
	Location   string `json:"location"`
}

type HttpsServer struct {
	HostName string `json:hostname`
	Location string `json:"location"`
}

type Servers struct {
	SS  []string
	Tls []string
}

func (tls *HttpsServer) ToString() string {
	return fmt.Sprintf("proxy = https://%s:%s@%s", UserInfo.Username, UserInfo.Password, strings.Trim(tls.HostName, "https://"))
}

func (dcs *DarkCloudServer) ToString() string {
	return fmt.Sprintf("proxy = ss://%s:%s@%s:%d", dcs.Encryption, dcs.Password, dcs.HostName, dcs.Port)
}

type Service struct {
	ServiceId int  `json:"id"`
	Disabled  int  `json:"disabled"`
	Plan      Plan `json:"plan"`
}

type Plan struct {
	Type string `json:"type"`
}

var client *http.Client

func getMonoCloudServers() (*Servers, error) {

	var err error
	serviceId := -1

	for i := 0; i < 2; i++ {
		err = getAccessToken(UserInfo.Username, UserInfo.Password, i != 0)
		if err != nil {
			return nil, err
		}
		serviceId, err = getServiceId()
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	var servers Servers
	var serr, terr error
	servers.SS, serr = getSsInfo(serviceId)
	servers.Tls, terr = getTlsInfo(serviceId)

	if serr != nil {
		err = serr
	}
	if terr != nil {
		err = terr
	}

	if len(servers.SS)+len(servers.Tls) == 0 {
		return &servers, err
	}

	return &servers, nil
}

//step 1 get access_token
func getAccessToken(username, password string, needNewToken bool) error {
	ctx := context.Background()
	clientId, clientSecret := getSecret()
	conf := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Endpoint:     MonoCloudEndpoint,
	}

	fs := NewFileCache()

	token, _ := fs.Read()

	tokenSource := conf.TokenSource(ctx, token)

	newToken, err := tokenSource.Token()

	if needNewToken || err != nil {

		if err != nil {
			log.Println(err)
		}
		newToken, err = conf.PasswordCredentialsToken(ctx, username, password)
		if err != nil {
			return err
		}
	}

	if token == nil || newToken.AccessToken != token.AccessToken {
		fs.Write(newToken)
		tokenSource = conf.TokenSource(ctx, newToken)
		log.Println("Saved new token:", newToken.AccessToken)
	}

	client = oauth2.NewClient(ctx, tokenSource)

	log.Println("Get access token success, expiry:", newToken.Expiry)

	return nil
}

//step 2 get serviceId
func getServiceId() (int, error) {
	var serviceId int = -1

	jsonData, err := getData(getUrl("service"))
	if err != nil {
		return serviceId, err
	}

	var services []Service
	err = json.Unmarshal(jsonData, &services)

	if err != nil {
		return serviceId, err
	}

	for _, service := range services {
		if service.Disabled == 0 && service.Plan.Type == "shadowsocks" {
			serviceId = service.ServiceId
			break
		}
	}

	if serviceId != -1 {
		log.Println("Get serviceId:", serviceId)
	}

	return serviceId, nil
}

func getSsInfo(serviceId int) ([]string, error) {

	jsonData, err := getData(getUrl("shadowsocks/" + strconv.Itoa(serviceId)))

	if err != nil {
		return nil, err
	}
	var servers []DarkCloudServer

	err = json.Unmarshal(jsonData, &servers)

	if err != nil {
		return nil, err
	}
	log.Println("Get Servers:", len(servers))

	var result []string
	for _, server := range servers {
		if server.Enable != 0 && server.SSR == 0 && server.Protocol == "origin" && !strings.Contains(server.Location, "中国大陆出口") {
			result = append(result, server.ToString())
		}
	}

	log.Println("OK Servers:", len(result))
	return result, nil
}

func getTlsInfo(serviceId int) ([]string, error) {

	jsonData, err := getData(getUrl("tls/" + strconv.Itoa(serviceId)))

	if err != nil {
		return nil, err
	}
	var servers []HttpsServer

	err = json.Unmarshal(jsonData, &servers)

	if err != nil {
		return nil, err
	}
	log.Println("Get Servers:", len(servers))

	var result []string
	for _, server := range servers {
		if !strings.Contains(server.Location, "中国大陆出口") {
			result = append(result, server.ToString())
		}
	}

	log.Println("OK Servers:", len(result))
	return result, nil
}

func getData(url string) ([]byte, error) {

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return body, err
}

func getUrl(method string) string {
	return fmt.Sprintf("%s%s", apiUrl, method)
}

func getSecret() (string, string) {
	return monoCloud_ClientId, monoCloud_ClientSecret
}
