package configparser

import (
	"bufio"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mattieserver/netbox-oxidized-sync/internal/model"
)

const (
	interfaceNamePrefix            = "    edit "
	interfaceTypePrefix            = "        set type "
	interfaceVlanIdPrefix          = "        set vlanid "
	interfaceParentInterfacePrefix = "        set interface "
	interfaceAliasPrefix           = "        set alias "
	interfaceVdomPrefix            = "        set vdom "
	interfaceIp                    = "        set ip "
	interfaceSpeed                 = "        set speed "
	intefaceMember                 = "        set member "
	interfaceStatus                = "        set status "
	interfaceDescription           = "        set description "
	virtualSwitchPortPrefix        = "            edit "
)

type FortiOSParser struct{}

func (FortiOSParser) Parse(config string) ([]model.ParsedInterface, error) {
	const (
		start              = "config system interface"
		end                = "end"
		startVirtualSwitch = "config system virtual-switch"
	)

	var (
		configVirtualSwitchTracking bool
		configInterfacesTracking    bool
		configInterfaces            []string
		configVirtualSwitch         []string
	)

	scanner := bufio.NewScanner(strings.NewReader(config))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case configInterfacesTracking && line == end:
			configInterfacesTracking = false
		case configVirtualSwitchTracking && line == end:
			configVirtualSwitchTracking = false
		case configVirtualSwitchTracking:
			configVirtualSwitch = append(configVirtualSwitch, line)
		case configInterfacesTracking:
			configInterfaces = append(configInterfaces, line)
		case line == start:
			configInterfacesTracking = true
		case line == startVirtualSwitch:
			configVirtualSwitchTracking = true
		}
	}

	deviceInterfaces := parseInterfaces(configInterfaces)
	deviceVirtualSwitches := parseVirtualSwitch(configVirtualSwitch)
	convertVirtualSwitch(deviceVirtualSwitches, &deviceInterfaces)

	// FortiGate-internal ports (modem, npu*) must never be created in NetBox, but
	// if an operator already tracks one we still keep it in sync. Mark instead of
	// drop so the generic differ updates an existing entry yet never creates one.
	// Likewise, an interface whose parent is an npu virtual-link points at a port
	// that is never synced, so it must be left untouched entirely.
	for i := range deviceInterfaces {
		if deviceInterfaces[i].Name == "modem" || strings.HasPrefix(deviceInterfaces[i].Name, "npu") {
			deviceInterfaces[i].NoCreate = true
		}
		if strings.HasPrefix(deviceInterfaces[i].Parent, "npu") {
			deviceInterfaces[i].NoUpdate = true
		}
	}

	return deviceInterfaces, nil
}

func parseVirtualSwitch(virtualSwitches []string) *[]model.FortigateVirtualSwitch {

	var deviceVirtualSwitches []model.FortigateVirtualSwitch

	var (
		configVirtualSwitch         []string
		configVirtualSwitchTracking bool
	)

	for _, element := range virtualSwitches {
		if strings.HasPrefix(element, interfaceNamePrefix) {
			configVirtualSwitchTracking = true
			configVirtualSwitch = []string{element}
			continue
		}

		if strings.HasPrefix(element, "    next") {
			if configVirtualSwitchTracking {
				configVirtualSwitchTracking = false
				parseSingleVirtualSwitch(configVirtualSwitch, &deviceVirtualSwitches)
			}
			continue
		}

		if configVirtualSwitchTracking {
			configVirtualSwitch = append(configVirtualSwitch, element)
		}
	}
	return &deviceVirtualSwitches
}

func parseSingleVirtualSwitch(virtualSwitchData []string, results *[]model.FortigateVirtualSwitch) {
	var name string
	var portNames []string

	portPrefixes := map[string]*[]string{
		virtualSwitchPortPrefix: &portNames,
	}

	prefixes := map[string]*string{
		interfaceNamePrefix: &name,
	}

	for _, element := range virtualSwitchData {
		for prefix, value := range prefixes {
			if strings.HasPrefix(element, prefix) {
				*value = getElementValue(element, prefix)
			}
		}

		for portPrefix, portValue := range portPrefixes {
			if strings.HasPrefix(element, portPrefix) {
				*portValue = append(*portValue, getElementValue(element, portPrefix))
			}
		}
	}

	if name == "''" {
		return
	}

	var vSwitch model.FortigateVirtualSwitch
	vSwitch.Name = name
	vSwitch.Members = portNames
	*results = append(*results, vSwitch)
}

func parseInterfaces(interfaces []string) []model.ParsedInterface {

	var deviceInterfaces []model.ParsedInterface

	var (
		configInterface         []string
		configInterfaceTracking bool
	)

	for _, element := range interfaces {

		if strings.HasPrefix(element, interfaceNamePrefix) {
			configInterfaceTracking = true
			configInterface = []string{element}
			continue
		}

		if strings.HasPrefix(element, "    next") {
			if configInterfaceTracking {
				configInterfaceTracking = false
				parseSingleInterface(configInterface, &deviceInterfaces)
			}
			continue
		}

		if configInterfaceTracking {
			configInterface = append(configInterface, element)
		}
	}

	return deviceInterfaces
}

func getElementValue(element string, filter string) string {
	return strings.ReplaceAll(strings.ReplaceAll(element, filter, ""), "\"", "")
}

func convertVirtualSwitch(virtutalSwitches *[]model.FortigateVirtualSwitch, deviceInterfaces *[]model.ParsedInterface) {

	var virtualSwitchNames = map[string]string{}

	for _, member := range *virtutalSwitches {
		var vswitch model.ParsedInterface
		vswitch.Name = member.Name
		vswitch.InterfaceType = model.TypeVirtualSwitch
		vswitch.Description = "virtual-switch"
		vswitch.Members = member.Members
		*deviceInterfaces = append(*deviceInterfaces, vswitch)
		for _, vswitchMember := range member.Members {
			virtualSwitchNames[vswitchMember] = member.Name
		}
	}

	for index, dinterface := range *deviceInterfaces {
		if virtualSwitchNames[dinterface.Name] != "" {
			(*deviceInterfaces)[index].Parent = virtualSwitchNames[dinterface.Name]
		}
	}
}

// fortiStatus maps a FortiOS "set status" value to the normalized tri-state.
// FortiOS only emits "set status down" for a disabled interface; an interface
// that is administratively up has no status line at all, so an empty value must
// map to StatusUp (not StatusUnknown) or up-but-disabled ports are never
// re-enabled in NetBox.
func fortiStatus(status string) model.InterfaceStatus {
	switch status {
	case "down":
		return model.StatusDown
	default:
		return model.StatusUp
	}
}

func parseSingleInterface(interfaceData []string, results *[]model.ParsedInterface) {

	var name, interfaceType, vlanId, parentName, alias, vdom, ip, speed, member, status, description string

	prefixes := map[string]*string{
		interfaceNamePrefix:            &name,
		interfaceTypePrefix:            &interfaceType,
		interfaceVlanIdPrefix:          &vlanId,
		interfaceParentInterfacePrefix: &parentName,
		interfaceAliasPrefix:           &alias,
		interfaceVdomPrefix:            &vdom,
		interfaceIp:                    &ip,
		interfaceSpeed:                 &speed,
		intefaceMember:                 &member,
		interfaceStatus:                &status,
		interfaceDescription:           &description,
	}

	for _, element := range interfaceData {
		for prefix, value := range prefixes {
			if strings.HasPrefix(element, prefix) {
				*value = getElementValue(element, prefix)
			}
		}
	}

	if name == "''" {
		return
	}

	if alias == "''" {
		alias = ""
	}

	switch interfaceType {
	case "aggregate", "redundant":
		var aggr model.ParsedInterface
		aggr.InterfaceType = model.TypeAggregate
		aggr.Name = name
		memberNames := strings.Split(member, " ")
		for _, memberName := range memberNames {
			if memberName != "" && memberName != "''" {
				aggr.Members = append(aggr.Members, memberName)
			}
		}
		aggr.Description = createDescription(alias, vdom, description)
		if interfaceType == "redundant" {
			aggr.Description = "redundant; " + aggr.Description
		}
		aggr.Status = fortiStatus(status)
		*results = append(*results, aggr)
	case "physical":
		var pyh model.ParsedInterface
		pyh.InterfaceType = model.TypePhysical
		pyh.Name = name
		pyh.Speed = speed
		pyh.Status = fortiStatus(status)
		pyh.Description = createDescription(alias, vdom, description)
		*results = append(*results, pyh)
	case "vlan":
		*results = append(*results, createVlan(name, alias, vdom, vlanId, parentName, description))
	case "loopback":
		slog.Warn("loopback interface; todo")
	case "":
		if vlanId != "" {
			*results = append(*results, createVlan(name, alias, vdom, vlanId, parentName, description))
		}
	}
}

func createVlan(name string, alias string, vdom string, vlanId string, parentName string, description string) model.ParsedInterface {
	var vid model.ParsedInterface
	vid.InterfaceType = model.TypeVlan
	if alias != "" {
		vid.Name = alias
	} else {
		vid.Name = name
	}
	vid.Description = createDescription(alias, vdom, description)
	vid.VlanId = vlanId
	vid.Parent = parentName
	return vid
}

func createDescription(alias string, vdom string, description string) string {
	var result string
	if vdom != "" {
		createDescriptionBuilder(vdom, "vdom", &result)
	}
	if alias != "" {
		createDescriptionBuilder(alias, "alias", &result)
	}
	if description != "" {
		createDescriptionBuilder(description, "desc", &result)
	}

	return result
}

func createDescriptionBuilder(value string, name string, result *string) {
	if *result != "" {
		*result = fmt.Sprintf("%s; %s: %s", *result, name, value)
	} else {
		*result = fmt.Sprintf("%s: %s", name, value)
	}
}
