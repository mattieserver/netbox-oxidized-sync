package netboxparser

import (
	"strconv"
	"strings"

	"github.com/mattieserver/netbox-oxidized-sync/internal/model"
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
			}

			if port.InterfaceType == "aggregate" && netboxInterface.Type.Value != "lag" {
				matched.PortTypeUpdate = "lag"
			}
			if !strings.EqualFold(port.Description, netboxInterface.Description) {
				matched.Description = port.Description
			}
			if port.InterfaceType == "physical" && len(allMembers) > 0 {
				if parentIndex, ok := allMembers[port.Name]; ok {
					if (*fortiInterfaces)[parentIndex].InterfaceType == "aggregate" {
						if netboxInterface.Lag.ID == 0 {
							matched.Parent = (*fortiInterfaces)[parentIndex].Name
						} else {
							if !strings.EqualFold(netboxInterface.Lag.Name, (*fortiInterfaces)[parentIndex].Name) {
								matched.Parent = (*fortiInterfaces)[parentIndex].Name
							}
						}
					} else if (*fortiInterfaces)[parentIndex].InterfaceType == "bridge" {
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
					matched.Parent = port.Parent
					matched.ParentId = getParentID(matched.Parent, netboxDeviceInterfaces)
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

	if matched == (model.NetboxInterfaceUpdateCreate{}) {
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
		} else if port.InterfaceType == "aggregate" && len(port.Members) > 0 {
			if len(port.Members) != 1 && port.Members[0] != "" {
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
		}
	} else {
		if matched.Description != "" || matched.Status != "" || matched.PortTypeUpdate != "" || matched.Parent != "" || matched.VlanMode != "" {
			matched.Mode = "update"
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
