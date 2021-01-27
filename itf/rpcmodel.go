package itf

import "github.com/mdzio/go-hmccu/itf/xmlrpc"

// DeviceDescription describes a HomeMatic device.
type DeviceDescription struct {
	Type              string
	Address           string
	RFAddress         int
	Children          []string
	Parent            string
	ParentType        string
	Index             int
	AESActive         int
	Paramsets         []string
	Firmware          string
	AvailableFirmware string
	Version           int

	// Flags is a bit mask for the presentation in the UI.
	// 0x01: visible for user
	// 0x02: internal (not visible)
	// 0x08: object not deleteable
	Flags int

	LinkSourceRoles string
	LinkTargetRoles string

	// Direction of a direct channel connection.
	// 0: none (direct connection not supported)
	// 1: sender
	// 2: receiver
	Direction int

	Group        string
	Team         string
	TeamTag      string
	TeamChannels []string
	Interface    string
	Roaming      int

	// RXMode is a bit mask of the receive modes.
	// 0x01: always
	// 0x02: burst (wake on radio)
	// 0x04: config (reachable after pressing config button)
	// 0x08: wakeup (after communication with the CCU)
	// 0x10: lazy config (config mode after normal use, e.g. key press)
	RXMode int
}

// ReadFrom reads the field values from an xmlrpc.Query.
func (d *DeviceDescription) ReadFrom(e *xmlrpc.Query) {
	d.Type = e.TryKey("TYPE").String()
	d.Address = e.TryKey("ADDRESS").String()
	d.RFAddress = e.TryKey("RF_ADDRESS").Int()
	// The interface VirtualDevices of the CCU returns an empty XML-RPC value
	// instead of an empty XML-RPC array, if the device has no children.
	c := e.TryKey("CHILDREN")
	if c.IsNotEmpty() {
		// If not empty, it must be an array of strings.
		d.Children = c.Strings()
	}
	d.Parent = e.TryKey("PARENT").String()
	d.ParentType = e.TryKey("PARENT_TYPE").String()
	d.Index = e.TryKey("INDEX").Int()
	d.AESActive = e.TryKey("AES_ACTIVE").Int()
	d.Paramsets = e.TryKey("PARAMSETS").Strings()
	d.Firmware = e.TryKey("FIRMWARE").String()
	d.AvailableFirmware = e.TryKey("AVAILABLE_FIRMWARE").String()
	d.Version = e.TryKey("VERSION").Int()
	d.Flags = e.TryKey("FLAGS").Int()
	d.LinkSourceRoles = e.TryKey("LINK_SOURCE_ROLES").String()
	d.LinkTargetRoles = e.TryKey("LINK_TARGET_ROLES").String()
	d.Direction = e.TryKey("DIRECTION").Int()
	d.Group = e.TryKey("GROUP").String()
	d.Team = e.TryKey("TEAM").String()
	d.TeamTag = e.TryKey("TEAM_TAG").String()
	d.TeamChannels = e.TryKey("TEAM_CHANNELS").Strings()
	d.Interface = e.TryKey("INTERFACE").String()
	d.Roaming = e.TryKey("ROAMING").Int()
	d.RXMode = e.TryKey("RX_MODE").Int()
}

// ParameterDescription describes a single parameter.
type ParameterDescription struct {
	// FLOAT, INTEGER, BOOL, ENUM, STRING, ACTION
	Type string

	// Bit field: 0x01=Read, 0x02=Write, 0x04=Event
	Operations int

	// Bit field: 0x01=Visible, 0x02=Internal, 0x04=Transform, 0x08=Service, 0x10=Sticky
	Flags int

	Default  interface{}
	Max      interface{}
	Min      interface{}
	Unit     string
	TabOrder int
	Control  string
	ID       string

	// Special parameters for Type=FLOAT:
	// ...

	// Special parameters for Type=INT:
	// ...

	// Special parameters for Type=ENUM:
	// ...
}

// ReadFrom reads the field values from an xmlrpc.Query.
func (p *ParameterDescription) ReadFrom(e *xmlrpc.Query) {
	p.Type = e.TryKey("TYPE").String()
	p.Operations = e.TryKey("OPERATIONS").Int()
	p.Flags = e.TryKey("FLAGS").Int()
	p.Default = e.TryKey("DEFAULT").Any()
	p.Min = e.TryKey("MIN").Any()
	p.Max = e.TryKey("MAX").Any()
	p.Unit = e.TryKey("UNIT").String()
	p.TabOrder = e.TryKey("TAB_ORDER").Int()
	p.Control = e.TryKey("CONTROL").String()
	p.ID = e.TryKey("ID").String()
}

// ParamsetDescription describes a parameter set (e.g. VALUES) of a device.
type ParamsetDescription map[string]*ParameterDescription
