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
	Description   string   `json:"description,omitempty"`
	Enabled       *bool    `json:"enabled,omitempty"`
	Parent        int      `json:"parent,omitempty"`
	Lag           int      `json:"lag,omitempty"`
	Bridge        int      `json:"bridge,omitempty"`
	InterfaceType string   `json:"type,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	UntaggedVlan  int      `json:"untagged_vlan,omitempty"`
	Tags          []string `json:"tags,omitempty"`
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

func NewNetbox(baseurl string, apikey string, roles []string) NetboxHTTPClient {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: customTransport}

	rolesfilter := ""
	if len(roles) > 0 {
		var sb strings.Builder
		
		for index, element := range roles {
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
		slog.Error("Error getting tags", "err", err)
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
		slog.Error("failed to create tag", "err", err)
	}

	var result model.NetboxTag
	err = json.Unmarshal(resBody, &result)
	if err != nil {
		slog.Error("failed to unmarshal tag", "err", err)
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

func (e *NetboxHTTPClient) GetIntefacesForDevice(deviceId string) ([]model.NetboxInterface, error) {
	requestURL := fmt.Sprintf("%s/api/dcim/interfaces/?device_id=%s", e.baseurl, deviceId)
	return apiRequest[model.NetboxInterface](requestURL, e)
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
		slog.Error("failed to create vlan", "err", err)
	}

	var result model.NetboxVlan
	err = json.Unmarshal(resBody, &result)
	if err != nil {
		slog.Error("failed to unmarshal vlan", "err", err)
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
			if port.PortType == model.TypePhysical {
				if port.ParentType == model.TypeVirtualSwitch {
					patchData.Bridge, _ = strconv.Atoi(port.ParentId)
				} else if port.ParentType == model.TypeAggregate {
					patchData.Lag, _ = strconv.Atoi(port.ParentId)
				}

			} else {
				patchData.Parent, _ = strconv.Atoi(port.ParentId)
			}

		} else {
			slog.Debug("parent interface does not exist yet", "interface", port.Name, "parent", port.Parent)
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
		if port.PortTypeUpdate == "bridge" {
			patchData.InterfaceType = "bridge"
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

	if len(port.Tags) != 0 {
		for _,tag := range port.Tags {
			if tag != strconv.Itoa(e.defaultTag.ID) {
				patchData.Tags = append(patchData.Tags, tag)
			}
		}
		patchData.Tags = append(patchData.Tags, strconv.Itoa(e.defaultTag.ID))
	} else {
		patchData.Tags = []string{strconv.Itoa(e.defaultTag.ID)}
	}

	data, _ := json.Marshal(patchData)
	requestURL := fmt.Sprintf("%s/%s%s/", e.baseurl, "api/dcim/interfaces/", port.InterfaceId)
	_, err := TokenAuthHTTPPatch(requestURL, e.apikey, &e.client, data)
	if err != nil {
		slog.Error("failed to update interface", "err", err)
	}

}

func (e *NetboxHTTPClient) createInterface(port model.NetboxInterfaceUpdateCreate, netboxVlansForSite *[]model.NetboxVlan, netboxSiteId int, netboxTenantId int) model.NetboxInterface {
	t := new(bool)
	f := new(bool)

	*t = true
	*f = false

	var postData interfacePostData
	postData.Name = port.Name
	postData.Device, _ = strconv.Atoi(port.DeviceId)

	if port.PortType == model.TypeAggregate {
		postData.InterfaceType = "lag"
	} else if port.PortType == model.TypeVlan {
		postData.InterfaceType = "virtual"
	} else if port.PortType == model.TypeVirtualSwitch {
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

	if port.PortType == model.TypePhysical {
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
			if port.PortType == model.TypePhysical {
				if port.ParentType == model.TypeVirtualSwitch {
					postData.Bridge, _ = strconv.Atoi(port.ParentId)
				} else if port.ParentType == model.TypeAggregate {
					postData.Lag, _ = strconv.Atoi(port.ParentId)
				}
			} else {
				postData.Parent, _ = strconv.Atoi(port.ParentId)
			}

		} else {
			slog.Debug("parent interface does not exist yet", "interface", port.Name, "parent", port.Parent)
		}
	}

	postData.Tags = []string{strconv.Itoa(e.defaultTag.ID)}

	data, _ := json.Marshal(postData)
	requestURL := fmt.Sprintf("%s/%s", e.baseurl, "api/dcim/interfaces/")
	resBody, err := TokenAuthHTTPPost(requestURL, e.apikey, &e.client, data)
	if err != nil {
		slog.Error("failed to create interface", "err", err)
		return model.NetboxInterface{}
	}

	var result model.NetboxInterface
	if err := json.Unmarshal(resBody, &result); err != nil {
		slog.Error("failed to unmarshal created interface", "err", err)
	}
	return result
}

func (e *NetboxHTTPClient) UpdateOrCreateInferface(interfaces *[]model.NetboxInterfaceUpdateCreate, netboxVlansForSite *[]model.NetboxVlan, netboxSiteId int, netboxTenantId int) {
	// Names referenced as a parent by some other interface. Any such interface
	// must be created before its children so the child can be linked, whatever
	// its type (a VLAN's parent can be a plain physical port, not just a LAG).
	parentNames := make(map[string]bool)
	for _, iface := range *interfaces {
		if iface.Parent != "" {
			parentNames[strings.ToLower(iface.Parent)] = true
		}
	}

	var parentInterfaces []model.NetboxInterfaceUpdateCreate
	var devicesWithParent []model.NetboxInterfaceUpdateCreate
	var standalone []model.NetboxInterfaceUpdateCreate

	for _, iface := range *interfaces {
		if iface.Parent != "" {
			devicesWithParent = append(devicesWithParent, iface)
			continue
		}

		if iface.PortType == model.TypeAggregate || iface.PortType == model.TypeVirtualSwitch || parentNames[strings.ToLower(iface.Name)] {
			parentInterfaces = append(parentInterfaces, iface)
			continue
		}
		standalone = append(standalone, iface)
	}

	createdParents := make(map[string]int)
	for _, port := range parentInterfaces {
		if port.Mode == "create" {
			created := e.createInterface(port, netboxVlansForSite, netboxSiteId, netboxTenantId)
			if created.ID != 0 {
				createdParents[strings.ToLower(port.Name)] = created.ID
			}
		}
		if port.Mode == "update" {
			e.updateInterface(port, netboxVlansForSite, netboxSiteId, netboxTenantId)
		}
	}

	for _, port := range devicesWithParent {
		if port.ParentId == "" && port.Parent != "" {
			if id, ok := createdParents[strings.ToLower(port.Parent)]; ok {
				port.ParentId = strconv.Itoa(id)
			}
		}
		if port.Mode == "create" {
			created := e.createInterface(port, netboxVlansForSite, netboxSiteId, netboxTenantId)
			if created.ID != 0 {
				createdParents[strings.ToLower(port.Name)] = created.ID
			}
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
