package configparser

import (

	"log"
	"strings"
)

func ParseFortiOSConfig(config *[]string) (error) {
	//start := "config system interface"
	//end := "end\n"

	fullConfig := strings.Join(*config, "")
	log.Println(fullConfig)

	

	return nil
}
