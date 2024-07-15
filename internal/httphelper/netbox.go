package httphelper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type netboxResult struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
}

type NetboxHttpClient struct {
	apikey  string
	baseurl string
	client  http.Client
}

func NewNetbox(baseurl string, apikey string) NetboxHttpClient {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: customTransport}

	e := NetboxHttpClient{apikey, baseurl, *client}
	return e
}

func loopApiRequest(path string, e *NetboxHttpClient) netboxResult {
	resBody, err := TokenAuthHttpGet(path, e.apikey, &e.client)
	if err != nil {
		log.Println("Something went wrong during http request")
	}
	var data netboxResult
	json.Unmarshal(resBody, &data)
	if err != nil {
		log.Println("Error:", err)
	}
	return data
}

func apiRequest(path string, e *NetboxHttpClient) {
	//var netboxResult = []
	reached_all := false
	url := path
	for !reached_all {
		data := loopApiRequest(url, e)
		fmt.Println(data.Next)
		if data.Next != "" {
			url = data.Next
		} else {
			reached_all = true
		}

	}
}

func (e *NetboxHttpClient) GetAllDevices() {
	requestURL := fmt.Sprintf("%s/%s", e.baseurl, "api/dcim/devices/")
	apiRequest(requestURL, e)
}
