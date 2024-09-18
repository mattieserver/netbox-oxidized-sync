package httphelper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
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
	Description   string `json:"description,omitempty"`
	Enabled       *bool  `json:"enabled,omitempty"`
	Parent        int    `json:"parent,omitempty"`
	Lag           int    `json:"lag,omitempty"`
	Bridge        int    `json:"bridge,omitempty"`
	InterfaceType string `json:"type,omitempty"`
	Mode          string `json:"mode,omitempty"`
	UntaggedVlan  int    `json:"untagged_vlan,omitempty"`
}

type interfacePostData struct {
	Device        int      `json:"device"`
	Name          string   `json:"name"`
	InterfaceType string   `json:"type"`
	Description   string   `json:"description,omitempty"`
	Enabled       *bool    `json:"enabled,omitempty"`
	UntaggedVlan  int      `json:"untagged_vlan,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	Parent        int      `json:"parent,omitempty"`
	Bridge        int      `json:"bridge,omitempty"`
	Lag           int      `json:"lag,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

type vlanPostData struct {
	SiteId   int      `json:"site,omitempty"`
	TenantId int      `json:"tenant,omitempty"`
	VlanId   int      `json:"vid"`
	Name     string   `json:"name"`
	Tags     []string `json:"tags,omitempty"`
}

type tagPostData struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
}

type netboxData interface {
	model.NetboxInterface | model.NetboxDevice | model.NetboxVlan | model.NetboxTag
}

type NetboxHTTPClient struct {
	apikey      string
	baseurl     string
	client      http.Client
	rolesfilter string
	defaultTag  model.NetboxTag
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

	e := NetboxHTTPClient{apikey, baseurl, *client, rolesfilter, model.NetboxTag{}}
	return e
}

func (e *NetboxHTTPClient) GetManagedTag(tagName string) {
	tag, err := getNetboxTagByName(tagName, e)
	if err != nil {
		slog.Error("Error getting tags", err)
	}
	if tag.ID == 0 {
		newTag := e.createNetboxTag(tagName)
		e.defaultTag = newTag
	} else {
		e.defaultTag = tag
	}
}

func getNetboxTagByName(tagName string, e *NetboxHTTPClient) (model.NetboxTag, error) {
	requestURL := fmt.Sprintf("%s/api/extras/tags/", e.baseurl)
	tags, err := apiRequest[model.NetboxTag](requestURL, e)
	if err != nil {
		return model.NetboxTag{}, err
	}
	for _, iface := range tags {
		if iface.Name == tagName {
			return iface, nil
		}
	}
	return model.NetboxTag{}, nil
}

func slugify(input string) string {
	var result string

	result = strings.ToLower(input)
	result = strings.Replace(result, " ", "-", -1)

	return result
}

func (e *NetboxHTTPClient) createNetboxTag(tagName string) model.NetboxTag {
	var postData tagPostData
	postData.Name = tagName
	postData.Slug = slugify(tagName)
	postData.Description = "Auto generated tag to track objects created by the oxidized sync"
	postData.Color = "72599f"

	data, _ := json.Marshal(postData)
	requestURL := fmt.Sprintf("%s/api/extras/tags/", e.baseurl)
	resBody, err := TokenAuthHTTPPost(requestURL, e.apikey, &e.client, data)
	if err != nil {
		slog.Error(err.Error())
	}

	var result model.NetboxTag
	err = json.Unmarshal(resBody, &result)
	if err != nil {
		slog.Error(err.Error())
	}
	return result
}

func loopAPIRequest(path string, e *NetboxHTTPClient) (netboxResult, error) {
	resBody, err := TokenAuthHTTPGet(path, e.apikey, &e.client)
	if err != nil {
		return netboxResult{}, fmt.Errorf("Something went wrong during http reques: %s", err)
	}

	var data netboxResult
	err = json.Unmarshal(resBody, &data)
	if err != nil {
		return netboxResult{}, fmt.Errorf("Error during Unmarshal: %s", err)
	}
	return data, nil
}

func apiRequest[T netboxData](path string, e *NetboxHTTPClient) ([]T, error) {
	netboxResult := []T{}
	reachedAll := false
	url := path

	for !reachedAll {
		data, err := loopAPIRequest(url, e)
		if err != nil {
			return []T{}, err
		}

		var actualResult []T
		err = json.Unmarshal(data.Results, &actualResult)
		if err != nil {
			return []T{}, fmt.Errorf("Error during Unmarshal of sub type: %s", err)
		}

		netboxResult = append(netboxResult, actualResult...)

		if data.Next != "" {
			url = data.Next
		} else {
			reachedAll = true
		}
	}
	return netboxResult, nil
}

func (e *NetboxHTTPClient) GetAllDevices() []model.NetboxDevice {
	requestURL := fmt.Sprintf("%s/api/dcim/devices/", e.baseurl)
	if e.rolesfilter != "" {
		requestURL = fmt.Sprintf("%s%s", requestURL, e.rolesfilter)
	}
	devices, _ := apiRequest[model.NetboxDevice](requestURL, e)
	return devices
}

func (e *NetboxHTTPClient) GetIntefacesForDevice(deviceId string) []model.NetboxInterface {
	requestURL := fmt.Sprintf("%s/api/dcim/interfaces/?device_id=%s", e.baseurl, deviceId)
	devices, _ := apiRequest[model.NetboxInterface](requestURL, e)
	return devices
}

func (e *NetboxHTTPClient) GetVlansForSite(siteId string) ([]model.NetboxVlan, error) {
	requestURL := fmt.Sprintf("%s/api/ipam/vlans/?site_id=%s", e.baseurl, siteId)
	vlans, err := apiRequest[model.NetboxVlan](requestURL, e)
	if err != nil {
		return []model.NetboxVlan{}, err
	}
	return vlans, nil
}

func getNetboxVlanInternalID(vlans *[]model.NetboxVlan, vid int) int {
	for _, vlan := range *vlans {
		if vlan.Vid == vid {
			return vlan.ID
		}
	}
	return 0
}

func (e *NetboxHTTPClient) createVlan(SiteId int, TenantId int, VlanId int, Name string) model.NetboxVlan {
	var postData vlanPostData
	postData.Name = Name
	postData.SiteId = SiteId
	postData.VlanId = VlanId
	postData.TenantId = TenantId
	postData.Tags = []string{strconv.Itoa(e.defaultTag.ID)}

	data, _ := json.Marshal(postData)
	requestURL := fmt.Sprintf("%s/api/ipam/vlans/", e.baseurl)
	resBody, err := TokenAuthHTTPPost(requestURL, e.apikey, &e.client, data)
	if err != nil {
		slog.Error(err.Error())
	}

	var result model.NetboxVlan
	err = json.Unmarshal(resBody, &result)
	if err != nil {
		slog.Error(err.Error())
	}
	return result

}

func (e *NetboxHTTPClient) updateInterface(port model.NetboxInterfaceUpdateCreate, netboxVlansForSite *[]model.NetboxVlan, netboxSiteId int, netboxTenantId int) {
	t := new(bool)
	f := new(bool)

	*t = true
	*f = false

	var patchData interfacePatchData

	if port.Parent != "" {
		if port.ParentId != "" {
			if port.PortType == "physical" {
				if port.ParentType == "virtual-switch" {
					patchData.Bridge, _ = strconv.Atoi(port.ParentId)
				} else if port.ParentType == "aggregate" {
					patchData.Lag, _ = strconv.Atoi(port.ParentId)
				}

			} else {
				patchData.Parent, _ = strconv.Atoi(port.ParentId)
			}

		} else {
			if !strings.HasPrefix(port.Parent, "npu") {
				slog.Info("Parent interface does not exist yet ")
			}
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
	if port.VlanMode != "" {
		patchData.Mode = port.VlanMode
	}

	if port.VlanId != "" {
		vid, _ := strconv.Atoi(port.VlanId)
		netboxVlanId := getNetboxVlanInternalID(netboxVlansForSite, vid)
		if netboxVlanId != 0 {
			patchData.UntaggedVlan = netboxVlanId
		} else {
			vlan := e.createVlan(netboxSiteId, netboxTenantId, vid, port.Name)
			*netboxVlansForSite = append(*netboxVlansForSite, vlan)
		}
	}

	data, _ := json.Marshal(patchData)
	requestURL := fmt.Sprintf("%s/%s%s/", e.baseurl, "api/dcim/interfaces/", port.InterfaceId)
	_, err := TokenAuthHTTPPatch(requestURL, e.apikey, &e.client, data)
	if err != nil {
		slog.Error(err.Error())
	}

}

func (e *NetboxHTTPClient) createInterface(port model.NetboxInterfaceUpdateCreate, netboxVlansForSite *[]model.NetboxVlan, netboxSiteId int, netboxTenantId int) {
	t := new(bool)
	f := new(bool)

	*t = true
	*f = false

	var postData interfacePostData
	postData.Name = port.Name
	postData.Device, _ = strconv.Atoi(port.DeviceId)

	if port.PortType == "aggregate" {
		postData.InterfaceType = "lag"
	} else if port.PortType == "vlan" {
		postData.InterfaceType = "virtual"
	} else if port.PortType == "virtual-switch" {
		postData.InterfaceType = "bridge"
	}

	if port.VlanId != "" {
		vid, _ := strconv.Atoi(port.VlanId)
		netboxVlanId := getNetboxVlanInternalID(netboxVlansForSite, vid)
		if netboxVlanId != 0 {
			postData.UntaggedVlan = netboxVlanId
		} else {
			vlan := e.createVlan(netboxSiteId, netboxTenantId, vid, port.Name)
			*netboxVlansForSite = append(*netboxVlansForSite, vlan)
		}

		if port.VlanMode != "" {
			postData.Mode = port.VlanMode
		}
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

	if port.Parent != "" {
		if port.ParentId != "" {
			if port.PortType == "physical" {
				if port.ParentType == "virtual-switch" {
					postData.Bridge, _ = strconv.Atoi(port.ParentId)
				} else if port.ParentType == "aggregate" {
					postData.Lag, _ = strconv.Atoi(port.ParentId)
				}
			} else {
				postData.Parent, _ = strconv.Atoi(port.ParentId)
			}

		} else {
			if !strings.HasPrefix(port.Parent, "npu") {
				slog.Info("Parent interface does not exist yet ")
			}
		}
	}

	postData.Tags = []string{strconv.Itoa(e.defaultTag.ID)}

	data, _ := json.Marshal(postData)
	requestURL := fmt.Sprintf("%s/%s", e.baseurl, "api/dcim/interfaces/")
	_, err := TokenAuthHTTPPost(requestURL, e.apikey, &e.client, data)
	if err != nil {
		slog.Error(err.Error())

	}
}

func (e *NetboxHTTPClient) UpdateOrCreateInferface(interfaces *[]model.NetboxInterfaceUpdateCreate, netboxVlansForSite *[]model.NetboxVlan, netboxSiteId int, netboxTenantId int) {
	var devicesWithParent []model.NetboxInterfaceUpdateCreate
	var lagInterfaces []model.NetboxInterfaceUpdateCreate
	var standalone []model.NetboxInterfaceUpdateCreate

	for _, iface := range *interfaces {
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
			e.createInterface(port, netboxVlansForSite, netboxSiteId, netboxTenantId)
		}
		if port.Mode == "update" {
			e.updateInterface(port, netboxVlansForSite, netboxSiteId, netboxTenantId)
		}
	}

	for _, port := range devicesWithParent {
		if port.Mode == "create" {
			e.createInterface(port, netboxVlansForSite, netboxSiteId, netboxTenantId)
		}
		if port.Mode == "update" {
			e.updateInterface(port, netboxVlansForSite, netboxSiteId, netboxTenantId)
		}
	}

	for _, port := range standalone {

		if port.Mode == "create" {
			e.createInterface(port, netboxVlansForSite, netboxSiteId, netboxTenantId)
		}
		if port.Mode == "update" {
			e.updateInterface(port, netboxVlansForSite, netboxSiteId, netboxTenantId)
		}
	}
}
