package httphelper

import (
	"fmt"
	"io"
	"net/http"
)


func BasicAuthHttpGet(baseurl string, path string, basicauth string, client *http.Client) ([]byte, error) {
	requestURL := fmt.Sprintf("%s/%s", baseurl, path)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	req.Header.Add("Authorization", "Basic "+ basicauth)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	return httpGet(req, client)
}

func TokenAuthHttpGet(fullurl string, token string, client *http.Client) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, fullurl, nil)
	req.Header.Add("Authorization", "Token "+ token)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	return httpGet(req, client)
}

func httpGet(req *http.Request, client *http.Client) ([]byte, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %s", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %s", err)

	}
	if res.StatusCode == http.StatusOK {
		return resBody, nil
	}
	return nil, fmt.Errorf("http status was not 200")

}
