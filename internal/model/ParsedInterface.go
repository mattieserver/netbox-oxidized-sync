package model


type InterfaceStatus int

const (
	StatusUnknown InterfaceStatus = iota // field absent in config -> leave NetBox enabled-state alone
	StatusUp
	StatusDown
)

const (
	TypePhysical      = "physical"
	TypeAggregate     = "aggregate"
	TypeVlan          = "vlan"
	TypeVirtualSwitch = "virtual-switch"
)


type ParsedInterface struct {
	Name          string
	Members       []string // LAG / virtual-switch member port names
	Description   string
	Status        InterfaceStatus
	Speed         string
	VlanId        string // string to match NetboxInterfaceUpdateCreate.VlanId; "" == no vlan
	Parent        string
	InterfaceType string // one of the Type* consts above

	// NoCreate marks an interface that should be updated when it already exists
	// in NetBox but never created from scratch (e.g. vendor-internal ports such
	// as a FortiGate "modem" or "npu*" interface).
	NoCreate bool
	// NoUpdate marks an interface that must be left untouched when it already
	// exists in NetBox (e.g. a subinterface whose parent is a vendor-internal
	// link that is never synced, so touching it would corrupt NetBox state).
	NoUpdate bool
}
