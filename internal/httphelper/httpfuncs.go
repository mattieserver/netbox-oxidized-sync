package httphelper

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)


func BasicAuthHttpGet(baseurl string, path string, basicauth string, client http.Client) ([]byte, error) {
	requestURL := fmt.Sprintf("%s/%s", baseurl, path)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	req.Header.Add("Authorization", "Basic "+ basicauth)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not create request: %s", err))
	}

	return httpGet(req, client)
}

func httpGet(req *http.Request, client http.Client) ([]byte, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error making http request: %s", err))
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read response body: %s", err))

	}
	return resBody, nil
}
