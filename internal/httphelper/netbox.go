package httphelper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mattieserver/netbox-oxidized-sync/internal/model"
)

type netboxResult struct {
	Count    int             `json:"count"`
	Next     string          `json:"next"`
	Previous string          `json:"previous"`
	Results  json.RawMessage `json:"results"`
}

type interfacePatchData struct {
	Description string `json:"description,omitempty"`
	Enabled *bool        `json:"enabled,omitempty"`
}

type netboxData interface {
	model.NetboxInterface | model.NetboxDevice
}

type NetboxHTTPClient struct {
	apikey      string
	baseurl     string
	client      http.Client
	rolesfilter string
}

func NewNetbox(baseurl string, apikey string, roles string) NetboxHTTPClient {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: customTransport}

	rolesfilter := ""
	if roles != "" {
		var sb strings.Builder
		splitRoles := strings.Split(roles, ",")
		for index, element := range splitRoles {
			if index == 0 {
				sb.WriteString(fmt.Sprintf("?role=%s", element))
			} else {
				sb.WriteString(fmt.Sprintf("&role=%s", element))
			}
		}
		rolesfilter = sb.String()
	}

	e := NetboxHTTPClient{apikey, baseurl, *client, rolesfilter}
	return e
}

func loopAPIRequest(path string, e *NetboxHTTPClient) netboxResult {
	resBody, err := TokenAuthHTTPGet(path, e.apikey, &e.client)
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

func apiRequest[T netboxData](path string, e *NetboxHTTPClient) []T {
	netboxResult := []T{}
	reachedAll := false
	url := path

	for !reachedAll {
		data := loopAPIRequest(url, e)

		var actualResult []T
		json.Unmarshal(data.Results, &actualResult)

		netboxResult = append(netboxResult, actualResult...)

		if data.Next != "" {
			url = data.Next
		} else {
			reachedAll = true
		}
	}
	return netboxResult
}

func (e *NetboxHTTPClient) GetAllDevices() []model.NetboxDevice {
	requestURL := fmt.Sprintf("%s/%s", e.baseurl, "api/dcim/devices/")
	if e.rolesfilter != "" {
		requestURL = fmt.Sprintf("%s%s", requestURL, e.rolesfilter)
	}
	devices := apiRequest[model.NetboxDevice](requestURL, e)
	return devices
}

func (e *NetboxHTTPClient) GetIntefacesForDevice(deviceId string) []model.NetboxInterface {
	requestURL := fmt.Sprintf("%s/%s%s", e.baseurl, "api/dcim/interfaces/?device_id=", deviceId)
	devices := apiRequest[model.NetboxInterface](requestURL, e)
	return devices
}

func (e *NetboxHTTPClient) UpdateOrCreateInferface(interfaces []model.NetboxInterfaceUpdateCreate) {
	t := new(bool)
    f := new(bool)

	*t = true
    *f = false

	for _, port := range interfaces {
		if port.Mode == "update" {
			var patchData interfacePatchData
			if port.PortType != "vlan" {
				slog.Info(port.Name)
			}

			if port.Description != "" {
				if len(port.Description) >= 200{
					slog.Warn("Description is to lang")
				} else {
					patchData.Description = port.Description
				}
				
			}
			if port.Status != "" {
				if port.Status == "disabled" {
					patchData.Enabled = f
				}
				if port.Status == "enabled" {
					patchData.Enabled = t
				}
			}

			data,_ := json.Marshal(patchData)
			requestURL := fmt.Sprintf("%s/%s%s/", e.baseurl, "api/dcim/interfaces/", port.InterfaceId)
			_, err := TokenAuthHTTPPatch(requestURL, e.apikey, &e.client, data)
			if err != nil {
				slog.Error(err.Error())
			}
		}
	}
}
