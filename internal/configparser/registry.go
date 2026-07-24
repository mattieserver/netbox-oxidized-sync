package configparser


func DefaultRegistry() map[string]ConfigParser {
	return map[string]ConfigParser{
		"FortiOS": FortiOSParser{},
		"FTOS":    FTOSParser{},
		// add new device classes here
	}
}
