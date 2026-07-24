package main

import (
	"log/slog"
	"os"
	"slices"
	"strconv"

	"github.com/mattieserver/netbox-oxidized-sync/internal/confighelper"
	"github.com/mattieserver/netbox-oxidized-sync/internal/configparser"
	"github.com/mattieserver/netbox-oxidized-sync/internal/httphelper"
	"github.com/mattieserver/netbox-oxidized-sync/internal/model"
	"github.com/mattieserver/netbox-oxidized-sync/internal/netboxparser"
)

func worker(id int, jobs <-chan httphelper.OxidizedNode, results chan<- int, netboxdevices *[]model.NetboxDevice, oxidizedhttp *httphelper.OxidizedHTTPClient, netboxhttp *httphelper.NetboxHTTPClient, registry map[string]configparser.ConfigParser) {
	for j := range jobs {
		slog.Info("got oxidized device", "device", j.Name, "worker", id)

		idx := slices.IndexFunc(*netboxdevices, func(c model.NetboxDevice) bool { return c.Name == j.Name })
		if idx == -1 {
			slog.Warn("device not found in netbox", "device", j.Name)
		} else {
			slog.Info("device found in netbox", "device", j.Name)

			parser, ok := registry[j.Model]
			if !ok {
				slog.Warn("model not supported", "model", j.Model)
			} else {
				config, err := oxidizedhttp.GetNodeConfig(j.FullName)
				if err != nil {
					slog.Error("failed to fetch config from oxidized", "device", j.FullName, "err", err)
				} else {
					syncDevice(parser, config, (*netboxdevices)[idx], netboxhttp)
				}
			}
		}

		results <- id * 2
	}
}

func syncDevice(parser configparser.ConfigParser, config string, netboxDevice model.NetboxDevice, netboxhttp *httphelper.NetboxHTTPClient) {
	slog.Info("syncing device", "device", netboxDevice.Name, "model", netboxDevice.DeviceType.Model)

	parsedInterfaces, err := parser.Parse(config)
	if err != nil {
		slog.Error("failed to parse config", "device", netboxDevice.Name, "err", err)
		return
	}

	netboxInterfaceForDevice, err := netboxhttp.GetIntefacesForDevice(strconv.Itoa(netboxDevice.ID))
	if err != nil {
		slog.Error("failed to fetch interfaces from netbox", "device", netboxDevice.Name, "err", err)
		return
	}

	netboxVlansForSite, err := netboxhttp.GetVlansForSite(strconv.Itoa(netboxDevice.Site.ID))
	if err != nil {
		slog.Error("failed to fetch vlans from netbox", "device", netboxDevice.Name, "err", err)
		return
	}

	interfacesToUpdate := netboxparser.BuildInterfaceChanges(parsedInterfaces, &netboxInterfaceForDevice, strconv.Itoa(netboxDevice.ID))
	netboxhttp.UpdateOrCreateInferface(&interfacesToUpdate, &netboxVlansForSite, netboxDevice.Site.ID, netboxDevice.Tenant.ID)
}

func loadOxidizedDevices(oxidizedhttp *httphelper.OxidizedHTTPClient, netboxhttp *httphelper.NetboxHTTPClient, registry map[string]configparser.ConfigParser) {
	slog.Info("fetching all oxidized devices")
	nodes := oxidizedhttp.GetAllNodes()
	slog.Info("fetched all oxidized devices")

	slog.Info("fetching all netbox devices")
	devices := netboxhttp.GetAllDevices()
	slog.Info("fetched all netbox devices")

	jobs := make(chan httphelper.OxidizedNode, len(nodes))
	results := make(chan int, len(nodes))

	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results, &devices, oxidizedhttp, netboxhttp, registry)
	}

	for _, element := range nodes {
		jobs <- element
	}
	close(jobs)

	for a := 1; a <= len(nodes); a++ {
		<-results
	}

}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("starting Oxidized to Netbox sync")

	conf := confighelper.ReadConfig()
	slog.Info("using netbox", "url", conf.Netbox.BaseURL)
	slog.Info("using oxidized", "url", conf.Oxidized.BaseURL)

	netboxhttp := httphelper.NewNetbox(conf.Netbox.BaseURL, conf.Netbox.APIKey, conf.Netbox.Roles)
	oxidizedhttp := httphelper.NewOxidized(conf.Oxidized.BaseURL, conf.Oxidized.Username, conf.Oxidized.Password)

	netboxhttp.GetManagedTag(conf.Netbox.TagName)

	registry := configparser.DefaultRegistry()

	loadOxidizedDevices(&oxidizedhttp, &netboxhttp, registry)
}
