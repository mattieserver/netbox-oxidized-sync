package main

import (
	"log"
	"strconv"

	"github.com/mattieserver/netbox-oxidized-sync/internal/confighelper"
	"github.com/mattieserver/netbox-oxidized-sync/internal/httphelper"
)

func worker(id int, jobs <-chan httphelper.OxidizedNode, results chan<- int, netboxdevices *[]httphelper.NetboxDevices) {
	for j := range jobs {
		log.Printf("Got oxided divice on worker %s with %s", strconv.Itoa(id), j.Name)
		results <- id * 2
	}
}

func loadOxidizedDevices(oxidizedhttp httphelper.OxidizedHttpClient, netboxhttp httphelper.NetboxHttpClient) {
	log.Println("Starting to get all Oxidized Devices")
	nodes := oxidizedhttp.GetAllNodes()
	log.Println("Got all Oxidized Devices")

	log.Println("Starting to get all Netbox Devices")
	devices := netboxhttp.GetAllDevices()
	log.Println("Got all Netbox Devices")

	jobs := make(chan httphelper.OxidizedNode, len(nodes))
	results := make(chan int, len(nodes))

	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results, &devices)
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

	loadOxidizedDevices(oxidizedhttp, netboxhttp)
}
