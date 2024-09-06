package httphelper

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)


func BasicAuthHTTPGet(baseurl string, path string, basicauth string, client *http.Client) ([]byte, error) {
	requestURL := fmt.Sprintf("%s/%s", baseurl, path)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	req.Header.Add("Authorization", "Basic "+ basicauth)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	return httpDo(req, client)
}

func TokenAuthHTTPGet(fullurl string, token string, client *http.Client) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, fullurl, nil)
	req.Header.Add("Authorization", "Token "+ token)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	return httpDo(req, client)
}

func TokenAuthHTTPPost(fullurl string, token string, client *http.Client, jsonbody []byte) ([]byte, error) {
	bodyReader := bytes.NewReader(jsonbody)
	req, err := http.NewRequest(http.MethodPost, fullurl, bodyReader)
	req.Header.Add("Authorization", "Token "+ token)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	resbody, err := httpDo(req, client)
	if err != nil {
		return nil, err
	}
	return resbody, nil
}

func TokenAuthHTTPPatch(fullurl string, token string, client *http.Client, jsonbody []byte) ([]byte, error) {
	bodyReader := bytes.NewReader(jsonbody)
	req, err := http.NewRequest(http.MethodPatch, fullurl, bodyReader)
	req.Header.Add("Authorization", "Token "+ token)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	return httpDo(req, client)
}

func httpDo(req *http.Request, client *http.Client) ([]byte, error) {
	res, err := client.Do(req)
	
	if err != nil {
		return nil, fmt.Errorf("error making http request: %s", err)
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %s", err)
	}	

	if res.StatusCode == http.StatusOK {
		return resBody, nil
	} else {
		if req.Method == http.MethodPost && res.StatusCode == http.StatusCreated{
			return resBody, nil
		}
	}	
	return nil, fmt.Errorf("http status was not 200")

}


