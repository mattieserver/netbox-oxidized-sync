package main

import (
	"log"
	"log/slog"
	"slices"
	"strconv"

	"github.com/mattieserver/netbox-oxidized-sync/internal/confighelper"
	"github.com/mattieserver/netbox-oxidized-sync/internal/configparser"
	"github.com/mattieserver/netbox-oxidized-sync/internal/httphelper"
	"github.com/mattieserver/netbox-oxidized-sync/internal/model"
	"github.com/mattieserver/netbox-oxidized-sync/internal/netboxparser"
)

func worker(id int, jobs <-chan httphelper.OxidizedNode, results chan<- int, netboxdevices *[]model.NetboxDevice, oxidizedhttp *httphelper.OxidizedHTTPClient, netboxhttp *httphelper.NetboxHTTPClient) {
	for j := range jobs {
		log.Printf("Got oxided device: '%s' on worker %s",j.Name, strconv.Itoa(id), )

		idx := slices.IndexFunc(*netboxdevices, func(c model.NetboxDevice) bool { return c.Name == j.Name })
		if idx == -1 {
			log.Printf("Device: '%s' not found in netbox", j.Name)
		} else {
			log.Printf("Device: '%s' found in netbox", j.Name)
			config := oxidizedhttp.GetNodeConfig(j.FullName)

			switch j.Model {
			case "IOS":
				log.Println("IOS not supported for now")
			case "FortiOS":
				log.Printf("Device: '%s' has fortiOS", j.Name)
				fortigateInterfaces,_ := configparser.ParseFortiOSConfig(&config)
				var netboxDevice = (*netboxdevices)[idx]
				netboxInterfaceForDevice := netboxhttp.GetIntefacesForDevice(strconv.Itoa(netboxDevice.ID))
				netboxVlansForSite, err := netboxhttp.GetVlansForSite(strconv.Itoa(netboxDevice.Site.ID))
				if err != nil {
					continue
				}
				interfacesToUpdate := netboxparser.ParseFortigateInterfaces(fortigateInterfaces, &netboxInterfaceForDevice, strconv.Itoa(netboxDevice.ID))
				netboxhttp.UpdateOrCreateInferface(&interfacesToUpdate, &netboxVlansForSite, netboxDevice.Site.ID, netboxDevice.Tenant.ID)
				
			default:
				log.Printf("Model '%s' currently not supported", j.Model)
			}
		}

		results <- id * 2
	}
}

func loadOxidizedDevices(oxidizedhttp *httphelper.OxidizedHTTPClient, netboxhttp *httphelper.NetboxHTTPClient) {
	log.Println("Starting to get all Oxidized Devices")
	nodes := oxidizedhttp.GetAllNodes()
	log.Println("Got all Oxidized Devices")

	log.Println("Starting to get all Netbox Devices")
	devices := netboxhttp.GetAllDevices()
	log.Println("Got all Netbox Devices")

	jobs := make(chan httphelper.OxidizedNode, len(nodes))
	results := make(chan int, len(nodes))

	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results, &devices, oxidizedhttp, netboxhttp)
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
	log.Println("Starting Oxidized to Netbox sync")

	conf := confighelper.ReadConfig()
	log.Printf("Using Netbox: %s", conf.Netbox.BaseURL)
	log.Printf("Using Oxidized: %s", conf.Oxidized.BaseURL)

	netboxhttp := httphelper.NewNetbox(conf.Netbox.BaseURL, conf.Netbox.APIKey, conf.Netbox.Roles)
	oxidizedhttp := httphelper.NewOxidized(conf.Oxidized.BaseURL, conf.Oxidized.Username, conf.Oxidized.Password)

	loadOxidizedDevices(&oxidizedhttp, &netboxhttp)
}
