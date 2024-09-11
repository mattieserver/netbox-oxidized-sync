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

func ParseFortiOSConfig(config *string) (*[]model.FortigateInterface, error) {
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

	scanner := bufio.NewScanner(strings.NewReader(*config))
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
	convertVirtualSwitch(deviceVirtualSwitches, deviceInterfaces)

	return deviceInterfaces, nil
}

func parseVirtualSwitch(virtualSwitches []string) *[]model.FortigateVirtualSwitch{

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

func parseSingleVirtualSwitch(virtualSwitchData []string, results *[]model.FortigateVirtualSwitch)  {
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

func parseInterfaces(interfaces []string) *[]model.FortigateInterface {

	var deviceInterfaces []model.FortigateInterface

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

	return &deviceInterfaces
}

func getElementValue(element string, filter string) string {
	return strings.ReplaceAll(strings.ReplaceAll(element, filter, ""), "\"", "")
}

func convertVirtualSwitch(virtutalSwitches *[]model.FortigateVirtualSwitch, deviceInterfaces *[]model.FortigateInterface ) {

	var virtualSwitchNames = map[string]string{}

	for _, member := range *virtutalSwitches {
		var vswitch model.FortigateInterface
		vswitch.Name = member.Name
		vswitch.InterfaceType = "virtual-switch"
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

func parseSingleInterface(interfaceData []string, results *[]model.FortigateInterface) {

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
	case "aggregate":
		var aggr model.FortigateInterface
		aggr.InterfaceType = "aggregate"
		aggr.Name = name
		memberNames := strings.Split(member, " ")
		aggr.Members = append(aggr.Members, memberNames...)
		aggr.Description = createDescription(alias, vdom, description)
		aggr.Status = status
		*results = append(*results, aggr)
	case "physical":
		var pyh model.FortigateInterface
		pyh.InterfaceType = "physical"
		pyh.Name = name
		pyh.Speed = speed
		pyh.Status = status
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

func createVlan(name string, alias string, vdom string, vlanId string, parentName string, description string) model.FortigateInterface {
	var vid model.FortigateInterface
	vid.InterfaceType = "vlan"
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
