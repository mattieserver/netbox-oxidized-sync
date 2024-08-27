package model


type FortigateInterfaces struct {
	AggregationPorts []AggregationDeviceInterface
	PhysicalPorts []PhysicalDeviceInterface
	Vlans []VirtualDeviceInterface
}

type AggregationDeviceInterface struct {
	Name string
	Members []string
	Description string
}

type PhysicalDeviceInterface struct {
	Name string
	Speed string
	Description string
}

type VirtualDeviceInterface struct {
	Name string
	VlanId string
	Parent string
	Description string
}
