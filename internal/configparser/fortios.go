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
)

func ParseFortiOSConfig(config *string) (*[]model.FortigateInterface, error) {
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

func parseSingleInterface(interfaceData []string, results *[]model.FortigateInterface) {

	var name, interfaceType, vlanId, parentName, alias, vdom, ip, speed, member, status string

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
		var aggr model.FortigateInterface
		aggr.InterfaceType = "aggregate"
		aggr.Name = name
		memberNames := strings.Split(member, " ")
		aggr.Members = append(aggr.Members, memberNames...)
		aggr.Description = createDescription(alias, vdom)
		aggr.Status = status
		*results = append(*results, aggr)
	case "physical":
		var pyh model.FortigateInterface
		pyh.InterfaceType = "physical"
		pyh.Name = name
		pyh.Speed = speed
		pyh.Status = status
		pyh.Description = createDescription(alias, vdom)
		*results = append(*results, pyh)
	case "vlan":
		*results = append(*results, createVlan(name, alias, vdom, vlanId, parentName))
	case "loopback":
		slog.Warn("loopback interface; todo")
	case "":
		if vlanId != "" {
			*results = append(*results, createVlan(name, alias, vdom, vlanId, parentName))
		}
	}
}

func createVlan(name string, alias string, vdom string, vlanId string, parentName string) model.FortigateInterface {
	var vid model.FortigateInterface
	vid.InterfaceType = "vlan"
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
