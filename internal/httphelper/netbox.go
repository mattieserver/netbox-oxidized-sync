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

type NetboxInterface struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	Display string `json:"display"`
	Device  struct {
		ID          int    `json:"id"`
		URL         string `json:"url"`
		Display     string `json:"display"`
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"device"`
	Vdcs   []interface{} `json:"vdcs"`
	Module interface{}   `json:"module"`
	Name   string        `json:"name"`
	Label  string        `json:"label"`
	Type   struct {
		Value string `json:"value"`
		Label string `json:"label"`
	} `json:"type"`
	Enabled                     bool          `json:"enabled"`
	Parent                      interface{}   `json:"parent"`
	Bridge                      interface{}   `json:"bridge"`
	Lag                         interface{}   `json:"lag"`
	Mtu                         interface{}   `json:"mtu"`
	MacAddress                  interface{}   `json:"mac_address"`
	Speed                       interface{}   `json:"speed"`
	Duplex                      interface{}   `json:"duplex"`
	Wwn                         interface{}   `json:"wwn"`
	MgmtOnly                    bool          `json:"mgmt_only"`
	Description                 string        `json:"description"`
	Mode                        interface{}   `json:"mode"`
	RfRole                      interface{}   `json:"rf_role"`
	RfChannel                   interface{}   `json:"rf_channel"`
	PoeMode                     interface{}   `json:"poe_mode"`
	PoeType                     interface{}   `json:"poe_type"`
	RfChannelFrequency          interface{}   `json:"rf_channel_frequency"`
	RfChannelWidth              interface{}   `json:"rf_channel_width"`
	TxPower                     interface{}   `json:"tx_power"`
	UntaggedVlan                interface{}   `json:"untagged_vlan"`
	TaggedVlans                 []interface{} `json:"tagged_vlans"`
	MarkConnected               bool          `json:"mark_connected"`
	Cable                       interface{}   `json:"cable"`
	CableEnd                    string        `json:"cable_end"`
	WirelessLink                interface{}   `json:"wireless_link"`
	LinkPeers                   []interface{} `json:"link_peers"`
	LinkPeersType               interface{}   `json:"link_peers_type"`
	WirelessLans                []interface{} `json:"wireless_lans"`
	Vrf                         interface{}   `json:"vrf"`
	L2VpnTermination            interface{}   `json:"l2vpn_termination"`
	ConnectedEndpoints          interface{}   `json:"connected_endpoints"`
	ConnectedEndpointsType      interface{}   `json:"connected_endpoints_type"`
	ConnectedEndpointsReachable interface{}   `json:"connected_endpoints_reachable"`
	Tags                        []interface{} `json:"tags"`
	CustomFields                struct {
	} `json:"custom_fields"`
	Created          time.Time `json:"created"`
	LastUpdated      time.Time `json:"last_updated"`
	CountIpaddresses int       `json:"count_ipaddresses"`
	CountFhrpGroups  int       `json:"count_fhrp_groups"`
	Occupied         bool      `json:"_occupied"`
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

type netboxData interface {
	NetboxInterface | NetboxDevice
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

func (e *NetboxHTTPClient) GetAllDevices() []NetboxDevice {
	requestURL := fmt.Sprintf("%s/%s", e.baseurl, "api/dcim/devices/")
	if e.rolesfilter != "" {
		requestURL = fmt.Sprintf("%s%s", requestURL, e.rolesfilter)
	}
	devices := apiRequest[NetboxDevice](requestURL, e)
	return devices
}

func (e *NetboxHTTPClient) GetIntefacesForDevice(deviceId string) []NetboxInterface {
	requestURL := fmt.Sprintf("%s/%s%s", e.baseurl, "api/dcim/interfaces/?device_id=", deviceId)
	devices := apiRequest[NetboxInterface](requestURL, e)
	return devices
}