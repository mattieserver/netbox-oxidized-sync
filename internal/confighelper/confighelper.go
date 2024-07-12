package confighelper

import (
	"os"
	"log"
	"encoding/json"
)

type config struct {
	Netbox struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
	} `json:"netbox"`
	Oxidized struct {
		BaseURL  string `json:"base_url"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"oxidized"`
}

func ReadConfig() config {
    f, err := os.ReadFile("configs/settings.json")
    if err != nil {
        log.Println(err)
    }

    var data config
	json.Unmarshal([]byte(f), &data)

	return data
	
}