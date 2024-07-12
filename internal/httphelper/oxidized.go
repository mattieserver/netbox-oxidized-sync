package httphelper

import (
	"crypto/tls"
	"encoding/base64"
	"log"
	"net/http"

)

type OxidizedHttpClient struct {
	username string
	password string
	baseurl  string
	client   http.Client
}

func NewOxidized(baseurl string, username string, password string) OxidizedHttpClient {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: customTransport}

	e := OxidizedHttpClient{username, password, baseurl, *client}
	return e
}

func (e OxidizedHttpClient) basicAuth() string {
	auth := e.username + ":" + e.password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (e OxidizedHttpClient) GetAllNodes() {
	resBody, err := BasicAuthHttpGet(e.baseurl, "nodes?format=json", e.basicAuth(), e.client)
	if err != nil {
		log.Println("Something went wrong during http request")
	}
	log.Println(resBody)
}
