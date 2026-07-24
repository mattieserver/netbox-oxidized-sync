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

func processPort(port model.ParsedInterface, allMembers map[string]int, parsedInterfaces []model.ParsedInterface, netboxDeviceInterfaces *[]model.NetboxInterface, deviceId string) model.NetboxInterfaceUpdateCreate {
	var matched model.NetboxInterfaceUpdateCreate
	for _, netboxInterface := range *netboxDeviceInterfaces {

		if strings.EqualFold(port.Name, netboxInterface.Name) {
			matched = model.NetboxInterfaceUpdateCreate{
				DeviceId:    deviceId,
				Name:        port.Name,
				PortType:    port.InterfaceType,
				InterfaceId: strconv.Itoa(netboxInterface.ID),
				Matched:     true,
			}

			if len(netboxInterface.Tags) != 0 {
				for _, tag := range netboxInterface.Tags {
					matched.Tags = append(matched.Tags, strconv.Itoa(tag.ID))
				}
			}

			if port.InterfaceType == model.TypeAggregate && netboxInterface.Type.Value != "lag" {
				matched.PortTypeUpdate = "lag"
			}
			if port.InterfaceType == model.TypeVirtualSwitch && netboxInterface.Type.Value != "bridge" {
				matched.PortTypeUpdate = "bridge"
			}
			if !strings.EqualFold(port.Description, netboxInterface.Description) {
				matched.Description = port.Description
			}
			if port.InterfaceType == model.TypePhysical && len(allMembers) > 0 {
				if parentIndex, ok := allMembers[port.Name]; ok {
					if parsedInterfaces[parentIndex].InterfaceType == model.TypeAggregate {
						matched.ParentType = model.TypeAggregate
						if netboxInterface.Lag.ID == 0 {
							matched.Parent = parsedInterfaces[parentIndex].Name
						} else {
							if !strings.EqualFold(netboxInterface.Lag.Name, parsedInterfaces[parentIndex].Name) {
								matched.Parent = parsedInterfaces[parentIndex].Name
							}
						}
					} else if parsedInterfaces[parentIndex].InterfaceType == model.TypeVirtualSwitch {
						matched.ParentType = model.TypeVirtualSwitch
						if netboxInterface.Bridge.ID == 0 {
							matched.Parent = parsedInterfaces[parentIndex].Name
						} else {
							if !strings.EqualFold(netboxInterface.Bridge.Name, parsedInterfaces[parentIndex].Name) {
								matched.Parent = parsedInterfaces[parentIndex].Name
							}
						}
					}

					if matched.Parent != "" {
						matched.ParentId = getParentID(matched.Parent, netboxDeviceInterfaces)
					}

				}
			}
			if port.InterfaceType == model.TypeVlan {
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
			// Tri-state: only act on an explicit up/down from the config.
			// StatusUnknown means the config said nothing, so leave NetBox's
			// enabled-state as-is (see model.InterfaceStatus).
			if port.Status == model.StatusDown && netboxInterface.Enabled {
				matched.Status = "disabled"
			} else if port.Status == model.StatusUp && !netboxInterface.Enabled {
				matched.Status = "enabled"
			}
			break
		}
	}

	if !matched.Matched {
		// Vendor-internal ports (e.g. FortiGate modem/npu) are updated when they
		// already exist but must never be created from scratch.
		if port.NoCreate {
			return matched
		}
		if port.InterfaceType == model.TypePhysical {
			matched.Mode = "create"
			matched.Name = port.Name
			matched.DeviceId = deviceId
			matched.Description = port.Description
			matched.PortType = port.InterfaceType
			if port.Status != model.StatusUnknown {
				if port.Status == model.StatusDown {
					matched.Status = "disabled"
				} else {
					matched.Status = "enabled"
				}
			}
			// Link a newly-created member port to its parent LAG / virtual-switch
			// so the relationship is set on the first sync, not only on updates.
			if len(allMembers) > 0 {
				if parentIndex, ok := allMembers[port.Name]; ok {
					switch parsedInterfaces[parentIndex].InterfaceType {
					case model.TypeAggregate:
						matched.ParentType = model.TypeAggregate
						matched.Parent = parsedInterfaces[parentIndex].Name
					case model.TypeVirtualSwitch:
						matched.ParentType = model.TypeVirtualSwitch
						matched.Parent = parsedInterfaces[parentIndex].Name
					}
					if matched.Parent != "" {
						matched.ParentId = getParentID(matched.Parent, netboxDeviceInterfaces)
					}
				}
			}
		} else if port.InterfaceType == model.TypeAggregate && len(port.Members) > 0 {
			if len(port.Members) >= 1 && port.Members[0] != "" {
				matched.Mode = "create"
				matched.PortType = port.InterfaceType
				matched.Name = port.Name
				matched.DeviceId = deviceId
				matched.Description = port.Description
				if port.Status != model.StatusUnknown {
					if port.Status == model.StatusDown {
						matched.Status = "disabled"
					} else {
						matched.Status = "enabled"
					}
				}
			}
		} else if port.InterfaceType == model.TypeVlan {
			matched.Mode = "create"
			matched.Name = port.Name
			matched.Description = port.Description
			matched.PortType = port.InterfaceType
			matched.DeviceId = deviceId
			matched.VlanMode = "access"
			matched.VlanId = port.VlanId
			matched.Parent = port.Parent
			matched.ParentId = getParentID(matched.Parent, netboxDeviceInterfaces)
		} else if port.InterfaceType == model.TypeVirtualSwitch {
			matched.Mode = "create"
			matched.Name = port.Name
			matched.Description = port.Description
			matched.PortType = port.InterfaceType
			matched.DeviceId = deviceId
		}
	} else if !port.NoUpdate {
		// NoUpdate interfaces reference a vendor-internal parent that is never
		// synced; touching them would corrupt NetBox state, so leave them as-is.
		if matched.Description != "" || matched.Status != "" || matched.PortTypeUpdate != "" || matched.Parent != "" || matched.VlanMode != "" {
			matched.Mode = "update"
		}
	}
	return matched
}


func BuildInterfaceChanges(parsedInterfaces []model.ParsedInterface, netboxDeviceInterfaces *[]model.NetboxInterface, deviceId string) []model.NetboxInterfaceUpdateCreate {
	var results []model.NetboxInterfaceUpdateCreate

	allMembers := make(map[string]int)
	for i, aggPort := range parsedInterfaces {
		for _, member := range aggPort.Members {
			allMembers[member] = i
		}
	}

	for _, port := range parsedInterfaces {
		result := processPort(port, allMembers, parsedInterfaces, netboxDeviceInterfaces, deviceId)
		if result.Mode != "" {
			results = append(results, result)
		}
	}

	return results
}
