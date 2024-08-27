package configparser

import (
	"bufio"
	"fmt"
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
)

func ParseFortiOSConfig(config *string) (*model.FortigateInterfaces, error) {
	const (
		start = "config system interface"
		end   = "end"
	)

	var (
		configInterfacesTracking bool
		configInterfaces         []string
	)

	scanner := bufio.NewScanner(strings.NewReader(*config))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case configInterfacesTracking && line == end:
			configInterfacesTracking = false
		case configInterfacesTracking:
			configInterfaces = append(configInterfaces, line)
		case line == start:
			configInterfacesTracking = true
		}
	}

	deviceInterfaces := parseInterfaces(configInterfaces)

	return deviceInterfaces, nil
}

func parseInterfaces(interfaces []string) *model.FortigateInterfaces {

	var deviceInterfaces model.FortigateInterfaces

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

func parseSingleInterface(interfaceData []string, results *model.FortigateInterfaces) {

	var name, interfaceType, vlanId, parentName, alias, vdom, ip, speed, member string

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
	}

	for _, element := range interfaceData {
		for prefix, value := range prefixes {
			if strings.HasPrefix(element, prefix) {
				*value = getElementValue(element, prefix)
			}
		}
	}

	switch interfaceType {
	case "aggregate":
		var aggr model.AggregationDeviceInterface
		aggr.Name = name
		memberNames := strings.Split(member, " ")
		aggr.Members = append(aggr.Members, memberNames...)
		aggr.Description = createDescription(alias, vdom)
		results.AggregationPorts = append(results.AggregationPorts, aggr)
	case "physical":
		var pyh model.PhysicalDeviceInterface
		pyh.Name = name
		pyh.Speed = speed
		pyh.Description = createDescription(alias, vdom)
		results.PhysicalPorts = append(results.PhysicalPorts, pyh)
	case "vlan":
		results.Vlans = append(results.Vlans, createVlan(name, alias, vdom, vlanId, parentName))
	case "":
		if vlanId != "" {
			results.Vlans = append(results.Vlans, createVlan(name, alias, vdom, vlanId, parentName))
		}
	}
}

func createVlan(name string, alias string, vdom string, vlanId string, parentName string) model.VirtualDeviceInterface {
	var vid model.VirtualDeviceInterface
	vid.Name = name
	vid.Description = createDescription(alias, vdom)
	vid.VlanId = vlanId
	vid.Parent = parentName
	return vid
}

func createDescription(alias string, vdom string) string {
	var result string
	if vdom != "" {
		createDescriptionBuilder(vdom, "vdom", &result)
	}
	if alias != "" {
		createDescriptionBuilder(alias, "alias", &result)
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


