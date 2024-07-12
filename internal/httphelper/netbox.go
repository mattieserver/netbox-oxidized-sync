package httphelper

import (
	"crypto/tls"
	"net/http"
)

type NetboxHttpClient struct {
	apikey string
	baseurl  string
	client   http.Client
}


func NewNetbox(apikey string, baseurl string) NetboxHttpClient{
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: customTransport}

	e := NetboxHttpClient{apikey, baseurl, *client}
	return e
}