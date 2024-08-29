package netboxparser

import (
	"strconv"
	"strings"

	"github.com/mattieserver/netbox-oxidized-sync/internal/model"
)



func processPort(port model.FortigateInterface, allMembers map[string]int, fortiInterfaces *[]model.FortigateInterface, netboxDeviceInterfaces *[]model.NetboxInterface) model.NetboxInterfaceUpdateCreate {
	var matched model.NetboxInterfaceUpdateCreate
	for _, netboxInterface := range *netboxDeviceInterfaces {
		if strings.EqualFold(port.Name, netboxInterface.Name) {
			matched = model.NetboxInterfaceUpdateCreate{
				DeviceId: strconv.Itoa(netboxInterface.Device.ID),
				Name:     port.Name,
				PortType: port.InterfaceType,
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
					if netboxInterface.Lag.ID == 0 {
						matched.Parent = (*fortiInterfaces)[parentIndex].Name
					} else {
						if !strings.EqualFold(netboxInterface.Lag.Name, (*fortiInterfaces)[parentIndex].Name) {
							matched.Parent = (*fortiInterfaces)[parentIndex].Name
						}
					}
				}
			}
			if port.InterfaceType == "vlan" {
				if port.Parent != "" {
					matched.Parent = port.Parent
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
		if port.InterfaceType == "phy" && port.Name != "modem" && !strings.HasPrefix(port.Name, "npu") {
			matched.Mode = "create"
		} else if port.InterfaceType == "agg" && len(port.Members) > 0 {
			matched.Mode = "create"
		}
	} else {
		if matched.Description != "" || matched.Status != "" || matched.PortTypeUpdate != "" || matched.Parent != "" || matched.VlanMode != "" {
			matched.Mode = "update"
		}
	}
	return matched
}

func ParseFortigateInterfaces(fortiInterfaces *[]model.FortigateInterface, netboxDeviceInterfaces *[]model.NetboxInterface)  []model.NetboxInterfaceUpdateCreate{
	var results []model.NetboxInterfaceUpdateCreate

	allMembers := make(map[string]int)
	for i, aggPort := range *fortiInterfaces {
		for _, member := range aggPort.Members {
			allMembers[member] = i
		}
	}

	for _, port := range *fortiInterfaces {
		result := processPort(port, allMembers, fortiInterfaces, netboxDeviceInterfaces)
		if result.Mode != "" {
			results = append(results, result)
		}
	}

	return results
}
