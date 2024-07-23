package httphelper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type netboxResult struct {
	Count    int             `json:"count"`
	Next     string          `json:"next"`
	Previous string          `json:"previous"`
	Results  json.RawMessage `json:"results"`
}

type NetboxDevice struct {
	ID         int    `json:"id"`
	URL        string `json:"url"`
	Display    string `json:"display"`
	Name       string `json:"name"`
	DeviceType struct {
		ID           int    `json:"id"`
		URL          string `json:"url"`
		Display      string `json:"display"`
		Manufacturer struct {
			ID      int    `json:"id"`
			URL     string `json:"url"`
			Display string `json:"display"`
			Name    string `json:"name"`
			Slug    string `json:"slug"`
		} `json:"manufacturer"`
		Model string `json:"model"`
		Slug  string `json:"slug"`
	} `json:"device_type"`
	Role struct {
		ID      int    `json:"id"`
		URL     string `json:"url"`
		Display string `json:"display"`
		Name    string `json:"name"`
		Slug    string `json:"slug"`
	} `json:"role"`
	DeviceRole struct {
		ID      int    `json:"id"`
		URL     string `json:"url"`
		Display string `json:"display"`
		Name    string `json:"name"`
		Slug    string `json:"slug"`
	} `json:"device_role"`
	Tenant struct {
		ID      int    `json:"id"`
		URL     string `json:"url"`
		Display string `json:"display"`
		Name    string `json:"name"`
		Slug    string `json:"slug"`
	} `json:"tenant"`
	Platform interface{} `json:"platform"`
	Serial   string      `json:"serial"`
	AssetTag interface{} `json:"asset_tag"`
	Site     struct {
		ID      int    `json:"id"`
		URL     string `json:"url"`
		Display string `json:"display"`
		Name    string `json:"name"`
		Slug    string `json:"slug"`
	} `json:"site"`
	Location     interface{} `json:"location"`
	Rack         interface{} `json:"rack"`
	Position     interface{} `json:"position"`
	Face         interface{} `json:"face"`
	Latitude     interface{} `json:"latitude"`
	Longitude    interface{} `json:"longitude"`
	ParentDevice interface{} `json:"parent_device"`
	Status       struct {
		Value string `json:"value"`
		Label string `json:"label"`
	} `json:"status"`
	Airflow        interface{} `json:"airflow"`
	PrimaryIP      interface{} `json:"primary_ip"`
	PrimaryIP4     interface{} `json:"primary_ip4"`
	PrimaryIP6     interface{} `json:"primary_ip6"`
	OobIP          interface{} `json:"oob_ip"`
	Cluster        interface{} `json:"cluster"`
	VirtualChassis interface{} `json:"virtual_chassis"`
	VcPosition     interface{} `json:"vc_position"`
	VcPriority     interface{} `json:"vc_priority"`
	Description    string      `json:"description"`
	Comments       string      `json:"comments"`
	ConfigTemplate interface{} `json:"config_template"`
	ConfigContext  struct {
	} `json:"config_context"`
	LocalContextData interface{}   `json:"local_context_data"`
	Tags             []interface{} `json:"tags"`
	CustomFields     struct {
		DeviceFirmwareFrequency   int         `json:"device_firmware_frequency"`
		DeviceFirmwareCurrent     interface{} `json:"device_firmware_current"`
		DeviceFirmwareLastupdate  interface{} `json:"device_firmware_lastupdate"`
		DeviceFirmwareRecommended interface{} `json:"device_firmware_recommended"`
		HACluster                 interface{} `json:"HA_Cluster"`
		Coordinates               interface{} `json:"coordinates"`
		DevicePowerUsage          interface{} `json:"device_power_usage"`
		SupportEndDate            interface{} `json:"support_end_date"`
	} `json:"custom_fields"`
	Created                time.Time `json:"created"`
	LastUpdated            time.Time `json:"last_updated"`
	ConsolePortCount       int       `json:"console_port_count"`
	ConsoleServerPortCount int       `json:"console_server_port_count"`
	PowerPortCount         int       `json:"power_port_count"`
	PowerOutletCount       int       `json:"power_outlet_count"`
	InterfaceCount         int       `json:"interface_count"`
	FrontPortCount         int       `json:"front_port_count"`
	RearPortCount          int       `json:"rear_port_count"`
	DeviceBayCount         int       `json:"device_bay_count"`
	ModuleBayCount         int       `json:"module_bay_count"`
	InventoryItemCount     int       `json:"inventory_item_count"`
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

func apiRequest[T NetboxDevice](path string, e *NetboxHTTPClient) []T {
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

func (e *NetboxHTTPClient) GetAllDevices() []NetboxDevice {
	requestURL := fmt.Sprintf("%s/%s", e.baseurl, "api/dcim/devices/")
	if e.rolesfilter != "" {
		requestURL = fmt.Sprintf("%s%s", requestURL, e.rolesfilter)
	}
	devices := apiRequest[NetboxDevice](requestURL, e)
	return devices
}
