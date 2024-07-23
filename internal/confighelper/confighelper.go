package confighelper

import (
	"os"
	"log"
	"encoding/json"
)

type Config struct {
	Netbox struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Roles	string `json:"roles"`
	} `json:"netbox"`
	Oxidized struct {
		BaseURL  string `json:"base_url"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"oxidized"`
}

func ReadConfig() Config {
    f, err := os.ReadFile("configs/settings.json")
    if err != nil {
        log.Println(err)
    }

    var data Config
	json.Unmarshal([]byte(f), &data)

	return data
	
}