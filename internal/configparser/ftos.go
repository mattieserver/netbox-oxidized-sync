package configparser

import (
	"bufio"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

const (
	interfaceFTOSNamePrefix = "interface "
)

func ParseFTOSConfig(config *string) error {
	const (
		interfaceStart = "interface "
		end            = "!"
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
			configInterfaces = append(configInterfaces, line)
		case configInterfacesTracking:
			configInterfaces = append(configInterfaces, line)
		case strings.HasPrefix(line, interfaceStart):
			if !strings.Contains(line, "interface breakout") {
				configInterfacesTracking = true
				configInterfaces = append(configInterfaces, line)
			}
		}
	}

	parseFTOSInterfaces(configInterfaces)

	return nil

}

func parseFTOSInterfaces(interfaces []string) {
	var (
		configInterface         []string
		configInterfaceTracking bool
	)

	for _, element := range interfaces {
		if strings.HasPrefix(element, interfaceFTOSNamePrefix) {
			configInterfaceTracking = true
			configInterface = []string{element}
			continue
		}

		if strings.HasPrefix(element, "!") {
			if configInterfaceTracking {
				configInterfaceTracking = false
				parseFTOSSingleInterface(configInterface)
			}
			continue
		}

		if configInterfaceTracking {
			configInterface = append(configInterface, element)
		}
	}
}

func parseFTOSSingleInterface(interfaceData []string) {

	interfaceEnabled := false
	var vlanId, channelGroupId, accessVlanID, mtu, interfaceChannelGroupId int
	var interfaceType, description, switchportMode, trunkVlans string

	for _, element := range interfaceData {
		if strings.HasPrefix(element, interfaceFTOSNamePrefix) {
			if strings.Contains(element, "interface vlan") {
				indexString, _ := strings.CutPrefix(element, "interface vlan")
				vlanId,_ = strconv.Atoi(indexString)
				interfaceType = "vlan"
			} else if strings.Contains(element, "interface port-channel") {
				indexString, _ := strings.CutPrefix(element, "interface port-channel")
				channelGroupId,_ = strconv.Atoi(indexString)
				interfaceType = "port-channel"
			} else if strings.Contains(element, "interface ethernet") {
				interfaceType = "ethernet"
			}
			continue
		}

		if strings.HasPrefix(element, " no shutdown") {
			interfaceEnabled = true
			continue
		}
		if strings.HasPrefix(element, " shutdown") {
			interfaceEnabled = false
			continue
		}

		if strings.HasPrefix(element, " description") {
			description, _ = strings.CutPrefix(element, " description ")
			continue
		}

		if strings.HasPrefix(element, " channel-group") {
			re := regexp.MustCompile(`channel-group\s+(\d+)`)
    		matches := re.FindStringSubmatch(element)
			if len(matches) > 1 {
				interfaceChannelGroupId,_ =  strconv.Atoi(matches[0])
			}
			continue
		}

		if strings.HasPrefix(element, " switchport mode") {
			switchportMode, _ = strings.CutPrefix(element, " switchport mode ")
			continue
		}

		if strings.HasPrefix(element, " switchport access vlan") {
			indexString, _ := strings.CutPrefix(element, " switchport access vlan ")
			accessVlanID,_ = strconv.Atoi(indexString)
			continue
		}

		if strings.HasPrefix(element, " switchport trunk allowed vlan") {
			trunkVlans, _ = strings.CutPrefix(element, " switchport trunk allowed vlan ")
			continue
		}

		if strings.HasPrefix(element, " mtu") {
			indexString, _ := strings.CutPrefix(element, " mtu  ")
			mtu,_ = strconv.Atoi(indexString)
			continue
		}
		

	}

	slog.Info("test", slog.Bool("interfaceEnabled", interfaceEnabled), slog.String("interfaceType", interfaceType), slog.Int("vlanid", vlanId),  slog.String("description", description), slog.Int("interfaceChannelGroupId", interfaceChannelGroupId), slog.Int("channelGroupID", channelGroupId), slog.String("switchportMode", switchportMode), slog.Int("accessVlanID ", accessVlanID), slog.String("trunkVlans", trunkVlans), slog.Int("mtu ", mtu), )
}
