package netboxparser

import (
	"strconv"
	"strings"

	"github.com/mattieserver/netbox-oxidized-sync/internal/model"
)

const (
	virtualSwitchName = "virtual-switch"
	lagName = "aggregate"
)

func getParentID(parentName string, netboxDeviceInterfaces *[]model.NetboxInterface) string {
	for _, netboxParentInterface := range *netboxDeviceInterfaces {
		if strings.EqualFold(netboxParentInterface.Name, parentName) {
			return strconv.Itoa(netboxParentInterface.ID)
		}
	}
	return ""
}

func processPort(port model.FortigateInterface, allMembers map[string]int, fortiInterfaces *[]model.FortigateInterface, netboxDeviceInterfaces *[]model.NetboxInterface, deviceId string) model.NetboxInterfaceUpdateCreate {
	var matched model.NetboxInterfaceUpdateCreate
	for _, netboxInterface := range *netboxDeviceInterfaces {

		if strings.EqualFold(port.Name, netboxInterface.Name) {
			matched = model.NetboxInterfaceUpdateCreate{
				DeviceId:    deviceId,
				Name:        port.Name,
				PortType:    port.InterfaceType,
				InterfaceId: strconv.Itoa(netboxInterface.ID),
				Matched : true,
			}

			if len(netboxInterface.Tags) != 0 {
				for _, tag := range netboxInterface.Tags {
					matched.Tags = append(matched.Tags,  strconv.Itoa(tag.ID))
				}
			}

			if port.InterfaceType == lagName && netboxInterface.Type.Value != "lag" {
				matched.PortTypeUpdate = "lag"
			}
			if port.InterfaceType == virtualSwitchName && netboxInterface.Type.Value != "bridge" {
				matched.PortTypeUpdate = "bridge"
			}
			if !strings.EqualFold(port.Description, netboxInterface.Description) {
				matched.Description = port.Description
			}
			if port.InterfaceType == "physical" && len(allMembers) > 0 {
				if parentIndex, ok := allMembers[port.Name]; ok {
					if (*fortiInterfaces)[parentIndex].InterfaceType == lagName {
						matched.ParentType = lagName
						if netboxInterface.Lag.ID == 0 {
							matched.Parent = (*fortiInterfaces)[parentIndex].Name
						} else {
							if !strings.EqualFold(netboxInterface.Lag.Name, (*fortiInterfaces)[parentIndex].Name) {
								matched.Parent = (*fortiInterfaces)[parentIndex].Name
							}
						}
					} else if (*fortiInterfaces)[parentIndex].InterfaceType == virtualSwitchName {
						matched.ParentType = virtualSwitchName
						if netboxInterface.Bridge.ID == 0 {
							matched.Parent = (*fortiInterfaces)[parentIndex].Name
						} else {
							if !strings.EqualFold(netboxInterface.Bridge.Name, (*fortiInterfaces)[parentIndex].Name) {
								matched.Parent = (*fortiInterfaces)[parentIndex].Name
							}
						}
					}

					if matched.Parent != "" {
						matched.ParentId = getParentID(matched.Parent, netboxDeviceInterfaces)
					}
					
				}
			}
			if port.InterfaceType == "vlan" {
				if port.Parent != "" {
					matched.ParentId = getParentID(port.Parent, netboxDeviceInterfaces)
					if matched.ParentId != strconv.Itoa(netboxInterface.Parent.ID) {
						matched.Parent = port.Parent
					}
				}

				if netboxInterface.Mode.Value != "access" {
					matched.VlanMode = "access"
				}
				if netboxInterface.Type.Value != "virtual" {
					matched.PortTypeUpdate = "virtual"
				}
				if port.VlanId != strconv.Itoa(netboxInterface.UntaggedVlan.Vid) {
					matched.VlanId = port.VlanId
				}
			}
			if port.Status != "" {
				if port.Status == "down" && netboxInterface.Enabled {
					matched.Status = "disabled"
				}
			} else if !netboxInterface.Enabled {
				matched.Status = "enabled"
			}
			break
		}
	}

	if !matched.Matched {
		if port.InterfaceType == "physical" && port.Name != "modem" && !strings.HasPrefix(port.Name, "npu") {
			matched.Mode = "create"
			matched.Name = port.Name
			matched.DeviceId = deviceId
			matched.Description = port.Description
			matched.PortType = port.InterfaceType
			if port.Status != "" {
				if port.Status == "down" {
					matched.Status = "disabled"
				} else {
					matched.Status = "enabled"
				}
			}
		} else if port.InterfaceType == lagName && len(port.Members) > 0 {
			if len(port.Members) >= 1 && port.Members[0] != "" {
				matched.Mode = "create"
				matched.PortType = port.InterfaceType
				matched.Name = port.Name
				matched.DeviceId = deviceId
				matched.Description = port.Description
				matched.Status = port.Status
				if port.Status != "" {
					if port.Status == "down" {
						matched.Status = "disabled"
					} else {
						matched.Status = "enabled"
					}
				}
			} 
		} else if port.InterfaceType == "vlan" {
			matched.Mode = "create"
			matched.Name = port.Name
			matched.Description = port.Description
			matched.PortType = port.InterfaceType
			matched.DeviceId = deviceId
			matched.VlanMode = "access"
			matched.VlanId = port.VlanId
			matched.Parent = port.Parent
			matched.ParentId = getParentID(matched.Parent, netboxDeviceInterfaces)
		} else if port.InterfaceType == virtualSwitchName {
			matched.Mode = "create"
			matched.Name = port.Name
			matched.Description = port.Description
			matched.PortType = port.InterfaceType
			matched.DeviceId = deviceId
		}
	} else {
		if matched.Description != "" || matched.Status != "" || matched.PortTypeUpdate != "" || matched.Parent != "" || matched.VlanMode != "" {
			if !strings.HasPrefix(port.Parent, "npu") {
				matched.Mode = "update"
			}			
		}
	}
	return matched
}

func ParseFortigateInterfaces(fortiInterfaces *[]model.FortigateInterface, netboxDeviceInterfaces *[]model.NetboxInterface, deviceId string) []model.NetboxInterfaceUpdateCreate {
	var results []model.NetboxInterfaceUpdateCreate

	allMembers := make(map[string]int)
	for i, aggPort := range *fortiInterfaces {
		for _, member := range aggPort.Members {
			allMembers[member] = i
		}
	}

	for _, port := range *fortiInterfaces {
		result := processPort(port, allMembers, fortiInterfaces, netboxDeviceInterfaces, deviceId)
		if result.Mode != "" {
			results = append(results, result)
		}
	}	

	return results
}
