package configparser

import (
	"bufio"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattieserver/netbox-oxidized-sync/internal/model"
)

const (
	interfaceFTOSNamePrefix = "interface "
)

var ftosChannelGroupRe = regexp.MustCompile(`channel-group\s+(\d+)`)

// ftosTrailingNumberRe extracts the numeric id at the end of an interface name,
// e.g. "100" from "Vlan 100" or "1" from "Port-channel 1".
var ftosTrailingNumberRe = regexp.MustCompile(`(\d+)\s*$`)

func ftosTrailingNumber(name string) int {
	m := ftosTrailingNumberRe.FindStringSubmatch(name)
	if len(m) > 1 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}


type FTOSParser struct{}

type ftosInterface struct {
	name           string
	interfaceType  string
	status         model.InterfaceStatus
	description    string
	vlanId         int 
	portChannelId  int 
	channelGroupId int 
}

func (FTOSParser) Parse(config string) ([]model.ParsedInterface, error) {
	const (
		interfaceStart = "interface "
		end            = "!"
	)

	var (
		configInterfacesTracking bool
		configInterfaces         []string
	)

	scanner := bufio.NewScanner(strings.NewReader(config))
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

	parsed := parseFTOSInterfaces(configInterfaces)
	return ftosToParsedInterfaces(parsed), nil
}

func parseFTOSInterfaces(interfaces []string) []ftosInterface {
	var (
		configInterface         []string
		configInterfaceTracking bool
		deviceInterfaces        []ftosInterface
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
				deviceInterfaces = append(deviceInterfaces, parseFTOSSingleInterface(configInterface))
			}
			continue
		}

		if configInterfaceTracking {
			configInterface = append(configInterface, element)
		}
	}

	return deviceInterfaces
}

func parseFTOSSingleInterface(interfaceData []string) ftosInterface {
	var iface ftosInterface
	iface.status = model.StatusUnknown

	for _, element := range interfaceData {
		if strings.HasPrefix(element, interfaceFTOSNamePrefix) {
			iface.name = strings.TrimSpace(strings.TrimPrefix(element, interfaceFTOSNamePrefix))
			// Match case-insensitively so both OS10 lowercase ("ethernet",
			// "port-channel", "vlan") and Force10/OS9 CamelCase names
			// ("GigabitEthernet", "TenGigabitEthernet", "FortyGigE",
			// "Port-channel", "Vlan") are recognized.
			lower := strings.ToLower(iface.name)
			switch {
			case strings.HasPrefix(lower, "vlan"):
				iface.interfaceType = model.TypeVlan
				iface.vlanId = ftosTrailingNumber(iface.name)
			case strings.HasPrefix(lower, "port-channel"):
				iface.interfaceType = model.TypeAggregate
				iface.portChannelId = ftosTrailingNumber(iface.name)
			case strings.HasPrefix(lower, "managementethernet"), strings.HasPrefix(lower, "mgmt"):
				// management port: never synced, leave type unset so it is skipped.
			case strings.Contains(lower, "ethernet"), strings.Contains(lower, "gige"):
				iface.interfaceType = model.TypePhysical
			}
			continue
		}

		if strings.HasPrefix(element, " no shutdown") {
			iface.status = model.StatusUp
			continue
		}
		if strings.HasPrefix(element, " shutdown") {
			iface.status = model.StatusDown
			continue
		}

		if strings.HasPrefix(element, " description") {
			iface.description, _ = strings.CutPrefix(element, " description ")
			continue
		}

		if strings.HasPrefix(element, " channel-group") {
			matches := ftosChannelGroupRe.FindStringSubmatch(element)
			if len(matches) > 1 {
				iface.channelGroupId, _ = strconv.Atoi(matches[1])
			}
			continue
		}
	}

	return iface
}


func ftosToParsedInterfaces(ifaces []ftosInterface) []model.ParsedInterface {
	portChannelIndex := map[int]int{} 

	var results []model.ParsedInterface
	for _, iface := range ifaces {
		if iface.interfaceType == "" || iface.name == "" {
			if iface.name != "" {
				slog.Warn("skipping unsupported FTOS interface", "name", iface.name)
			}
			continue
		}
		pi := model.ParsedInterface{
			Name:          iface.name,
			Description:   iface.description,
			Status:        iface.status,
			InterfaceType: iface.interfaceType,
		}
		if iface.interfaceType == model.TypeVlan {
			pi.VlanId = strconv.Itoa(iface.vlanId)
		}
		results = append(results, pi)
		if iface.interfaceType == model.TypeAggregate {
			portChannelIndex[iface.portChannelId] = len(results) - 1
		}
	}


	for i, iface := range ifaces {
		if iface.interfaceType != model.TypePhysical || iface.channelGroupId == 0 {
			continue
		}
		if pcIdx, ok := portChannelIndex[iface.channelGroupId]; ok {
			results[pcIdx].Members = append(results[pcIdx].Members, ifaces[i].name)
		}
	}

	return results
}
