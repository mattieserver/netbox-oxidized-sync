package configparser

import (
	"bufio"
	"log/slog"
	"strings"
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

func ParseFortiOSConfig(config *string) error {
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

	parseInterfaces(configInterfaces)

	return nil
}

func parseInterfaces(interfaces []string) {

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
				parseSingleInterface(configInterface)
			}
			continue
		}

		if configInterfaceTracking {
			configInterface = append(configInterface, element)
		}
	}
}

func getElementValue(element string, filter string) string {
	return strings.ReplaceAll(strings.ReplaceAll(element, filter, ""), "\"", "")
}

func parseSingleInterface(interfaceData []string) {

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

	slog.Info(name)
	slog.Info(interfaceType)
	if interfaceType == "vlan" {
		slog.Info("vlan")
	} else {
		if interfaceType == "aggregate" {
			slog.Info("agg")
		}
	}

	slog.Info(vlanId)
	slog.Info(parentName)
	slog.Info(alias)
	slog.Info(vdom)
	if speed != "" {
		slog.Info(speed)
	}

}
