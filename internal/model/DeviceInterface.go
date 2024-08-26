package model


type deviceInterface struct {
	Name string
	deviceType int
}

func (i *deviceInterface)SetType(dType string) {
	i.deviceType = TypeMapper(dType)
}

func NewDeviceInterface() deviceInterface {
	dInterface := deviceInterface{}
	dInterface.SetType("no")
	return dInterface
}

func TypeMapper(dType string) int {
	switch dType {
	case "vlan":
		return 1
	default:
		return 0
	}	
}

func TypeMapperReverse(dType int) string {
	switch dType {
	case 1:
		return "vlan"
	default:
		return "unkown"
	}	
}