package httphelper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
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
	Enabled     *bool  `json:"enabled,omitempty"`
	Parent      int    `json:"parent,omitempty"`
	Lag         int    `json:"lag,omitempty"`
	InterfaceType string `json:"type,omitempty"`
}

type interfacePostData struct {
	Device        int    `json:"device"`
	Name          string `json:"name"`
	InterfaceType string `json:"type"`
	Description   string `json:"description,omitempty"`
	Enabled       *bool  `json:"enabled,omitempty"`
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

func (e *NetboxHTTPClient) updateInterface(port model.NetboxInterfaceUpdateCreate) {
	t := new(bool)
	f := new(bool)

	*t = true
	*f = false

	var patchData interfacePatchData

	if port.Parent != "" {
		if port.ParentId != "" {
			if port.PortType == "physical" {
				patchData.Lag, _ = strconv.Atoi(port.ParentId)
			} else {
				patchData.Parent, _ = strconv.Atoi(port.ParentId)
			}

		} else {
			slog.Warn("todo")
		}
	}

	if port.Description != "" {
		if len(port.Description) >= 200 {
			slog.Warn("Description is to long")
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

	if port.PortTypeUpdate != "" {
		if port.PortTypeUpdate == "virtual" {
			patchData.InterfaceType = "virtual"
		}
	}

	data, _ := json.Marshal(patchData)
	requestURL := fmt.Sprintf("%s/%s%s/", e.baseurl, "api/dcim/interfaces/", port.InterfaceId)
	_, err := TokenAuthHTTPPatch(requestURL, e.apikey, &e.client, data)
	if err != nil {
		slog.Error(err.Error())
	}
}

func (e *NetboxHTTPClient) createInterface(port model.NetboxInterfaceUpdateCreate) {
	t := new(bool)
	f := new(bool)

	*t = true
	*f = false

	var postData interfacePostData

	postData.Name = port.Name
	postData.Device, _ = strconv.Atoi(port.DeviceId)

	if port.PortType == "aggregate" {
		postData.InterfaceType = "lag"
	}

	if port.PortType == "physical" {
		postData.InterfaceType = "1000base-t"
	}

	if port.Status != "" {
		if port.Status == "disabled" {
			postData.Enabled = f
		}
		if port.Status == "enabled" {
			postData.Enabled = t
		}
	}

	if port.Description != "" {
		if len(port.Description) >= 200 {
			slog.Warn("Description is to long")
		} else {
			postData.Description = port.Description
		}
	}

	data, _ := json.Marshal(postData)
	requestURL := fmt.Sprintf("%s/%s", e.baseurl, "api/dcim/interfaces/")
	_, err := TokenAuthHTTPPost(requestURL, e.apikey, &e.client, data)
	if err != nil {
		slog.Error(err.Error())
	}
}

func (e *NetboxHTTPClient) UpdateOrCreateInferface(interfaces []model.NetboxInterfaceUpdateCreate) {
	var devicesWithParent []model.NetboxInterfaceUpdateCreate
	var lagInterfaces []model.NetboxInterfaceUpdateCreate
	var standalone []model.NetboxInterfaceUpdateCreate

	for _, iface := range interfaces {
		if iface.Parent != "" {
			devicesWithParent = append(devicesWithParent, iface)
			continue
		}
		if iface.PortType == "aggregate" {
			lagInterfaces = append(lagInterfaces, iface)
			continue
		}
		standalone = append(standalone, iface)
	}

	for _, port := range lagInterfaces {
		if port.Mode == "create" {
			e.createInterface(port)
		}
		if port.Mode == "update" {
			e.updateInterface(port)
		}
	}

	for _, port := range devicesWithParent {
		if port.Mode == "create" {
			e.createInterface(port)
		}
		if port.Mode == "update" {
			e.updateInterface(port)
		}
	}

	for _, port := range standalone {
		if port.Mode == "create" {
			e.createInterface(port)
		}
		if port.Mode == "update" {
			e.updateInterface(port)
		}
	}
}
