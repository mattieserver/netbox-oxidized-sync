package configparser

import "github.com/mattieserver/netbox-oxidized-sync/internal/model"


type ConfigParser interface {
	Parse(config string) ([]model.ParsedInterface, error)
}
