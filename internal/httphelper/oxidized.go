package httphelper

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
)

type OxidizedNodes []OxidizedNode


type OxidizedNode struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	IP       string `json:"ip"`
	Group    string `json:"group"`
	Model    string `json:"model"`
	Last     struct {
		Start  string  `json:"start"`
		End    string  `json:"end"`
		Status string  `json:"status"`
		Time   float64 `json:"time"`
	} `json:"last"`
	Vars struct {
		SSHPort interface{} `json:"ssh_port"`
	} `json:"vars"`
	Mtime  string `json:"mtime"`
	Status string `json:"status"`
	Time   string `json:"time"`
}

type OxidizedHTTPClient struct {
	username string
	password string
	baseurl  string
	client   http.Client
}

func NewOxidized(baseurl string, username string, password string) OxidizedHTTPClient {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: customTransport}

	e := OxidizedHTTPClient{username, password, baseurl, *client}
	return e
}

func (e *OxidizedHTTPClient) basicAuth() string {
	auth := e.username + ":" + e.password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (e *OxidizedHTTPClient) GetAllNodes() OxidizedNodes {
	resBody, err := BasicAuthHTTPGet(e.baseurl, "nodes?format=json", e.basicAuth(), &e.client)
	if err != nil {
		log.Println("Something went wrong during http request")
	}

	var nodes OxidizedNodes	
	json.Unmarshal(resBody, &nodes)
	if err != nil {
		log.Println("Error:", err)
	}
	return nodes
}
