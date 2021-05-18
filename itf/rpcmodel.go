package itf

import (
	"fmt"
	"strings"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

const (
	DeviceFlagVisible = 1 << iota
	DeviceFlagInternal
	_
	DeviceFlagNotDeletable
)

const (
	DeviceDirectionNone = iota
	DeviceDirectionSender
	DeviceDirectionReceiver
)

const (
	DeviceRXModeAlways = 1 << iota
	DeviceRXModeBurst
	DeviceRXModeConfig
	DeviceRXModeWakeUp
	DeviceRXModeLazyConfig
)

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

// ToValue returns an xmlrpc.Value for this device description.
func (d *DeviceDescription) ToValue() *xmlrpc.Value {
	return &xmlrpc.Value{
		Struct: &xmlrpc.Struct{Members: []*xmlrpc.Member{
			{Name: "TYPE", Value: xmlrpc.NewString(d.Type)},
			{Name: "ADDRESS", Value: xmlrpc.NewString(d.Address)},
			{Name: "RF_ADDRESS", Value: xmlrpc.NewInt(d.RFAddress)},
			{Name: "CHILDREN", Value: xmlrpc.NewStrings(d.Children)},
			{Name: "PARENT", Value: xmlrpc.NewString(d.Parent)},
			{Name: "PARENT_TYPE", Value: xmlrpc.NewString(d.ParentType)},
			{Name: "INDEX", Value: xmlrpc.NewInt(d.Index)},
			{Name: "AES_ACTIVE", Value: xmlrpc.NewInt(d.AESActive)},
			{Name: "PARAMSETS", Value: xmlrpc.NewStrings(d.Paramsets)},
			{Name: "FIRMWARE", Value: xmlrpc.NewString(d.Firmware)},
			{Name: "AVAILABLE_FIRMWARE", Value: xmlrpc.NewString(d.AvailableFirmware)},
			{Name: "VERSION", Value: xmlrpc.NewInt(d.Version)},
			{Name: "FLAGS", Value: xmlrpc.NewInt(d.Flags)},
			{Name: "LINK_SOURCE_ROLES", Value: xmlrpc.NewString(d.LinkSourceRoles)},
			{Name: "LINK_TARGET_ROLES", Value: xmlrpc.NewString(d.LinkTargetRoles)},
			{Name: "DIRECTION", Value: xmlrpc.NewInt(d.Direction)},
			{Name: "GROUP", Value: xmlrpc.NewString(d.Group)},
			{Name: "TEAM", Value: xmlrpc.NewString(d.Team)},
			{Name: "TEAM_TAG", Value: xmlrpc.NewString(d.TeamTag)},
			{Name: "TEAM_CHANNELS", Value: xmlrpc.NewStrings(d.TeamChannels)},
			{Name: "INTERFACE", Value: xmlrpc.NewString(d.Interface)},
			{Name: "ROAMING", Value: xmlrpc.NewInt(d.Roaming)},
			{Name: "RX_MODE", Value: xmlrpc.NewInt(d.RXMode)},
		}},
	}
}

const (
	ParameterTypeFloat   = "FLOAT"
	ParameterTypeInteger = "INTEGER"
	ParameterTypeBool    = "BOOL"
	ParameterTypeEnum    = "ENUM"
	ParameterTypeString  = "STRING"
	ParameterTypeAction  = "ACTION"
)

const (
	ParameterOperationRead = 1 << iota
	ParameterOperationWrite
	ParameterOperationEvent
)

const (
	ParameterFlagVisible = 1 << iota
	ParameterFlagInternal
	ParameterFlagTransform
	ParameterFlagService
	ParameterFlagSticky
)

// SpecialValue defines a special value für an INTEGER or FLOAT. Value must be
// of type int or float64.
type SpecialValue struct {
	ID    string
	Value interface{}
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

	// Only for type FLOAT or INTEGER
	Special []SpecialValue

	// Only for type ENUM
	ValueList []string
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

	// read special properties
	switch p.Type {
	case "FLOAT":
		for _, s := range e.TryKey("SPECIAL").Slice() {
			id := s.Key("ID").String()
			val := s.Key("VALUE").Float64()
			p.Special = append(p.Special, SpecialValue{id, val})
		}
	case "INTEGER":
		for _, s := range e.TryKey("SPECIAL").Slice() {
			id := s.Key("ID").String()
			val := s.Key("VALUE").Int()
			p.Special = append(p.Special, SpecialValue{id, val})
		}
	case "ENUM":
		p.ValueList = e.TryKey("VALUE_LIST").Strings()
	}
}

// ToValue returns an xmlrpc.Value for this device description.
func (p *ParameterDescription) ToValue() (*xmlrpc.Value, error) {
	dflt, err := xmlrpc.NewValue(p.Default)
	if err != nil {
		return nil, err
	}
	min, err := xmlrpc.NewValue(p.Min)
	if err != nil {
		return nil, err
	}
	max, err := xmlrpc.NewValue(p.Max)
	if err != nil {
		return nil, err
	}

	v := &xmlrpc.Value{
		Struct: &xmlrpc.Struct{Members: []*xmlrpc.Member{
			{Name: "TYPE", Value: xmlrpc.NewString(p.Type)},
			{Name: "OPERATIONS", Value: xmlrpc.NewInt(p.Operations)},
			{Name: "FLAGS", Value: xmlrpc.NewInt(p.Flags)},
			{Name: "DEFAULT", Value: dflt},
			{Name: "MIN", Value: min},
			{Name: "MAX", Value: max},
			{Name: "UNIT", Value: xmlrpc.NewString(p.Unit)},
			{Name: "TAB_ORDER", Value: xmlrpc.NewInt(p.TabOrder)},
			{Name: "CONTROL", Value: xmlrpc.NewString(p.Control)},
			{Name: "ID", Value: xmlrpc.NewString(p.ID)},
		}},
	}

	// write special properties
	switch p.Type {
	case "FLOAT":
		fallthrough
	case "INTEGER":
		es := make([]*xmlrpc.Value, len(p.Special))
		for i := range p.Special {
			var sv *xmlrpc.Value
			switch ev := p.Special[i].Value.(type) {
			case float64:
				sv = xmlrpc.NewFloat64(ev)
			case int:
				sv = xmlrpc.NewInt(ev)
			default:
				return nil, fmt.Errorf("Expected type int or float64 for SPECIAL property of parameter description: %T", ev)
			}
			es[i] = &xmlrpc.Value{Struct: &xmlrpc.Struct{Members: []*xmlrpc.Member{
				{Name: "ID", Value: xmlrpc.NewString(p.Special[i].ID)},
				{Name: "VALUE", Value: sv},
			}}}
		}
		v.Struct.Members = append(v.Struct.Members, &xmlrpc.Member{
			Name: "SPECIAL", Value: &xmlrpc.Value{Array: &xmlrpc.Array{Data: es}},
		})
	case "ENUM":
		v.Struct.Members = append(v.Struct.Members, &xmlrpc.Member{
			Name: "VALUE_LIST", Value: xmlrpc.NewStrings(p.ValueList),
		})
	}
	return v, nil
}

// ParamsetDescription describes a parameter set (e.g. VALUES) of a device.
type ParamsetDescription map[string]*ParameterDescription

// ReadFrom reads the field values from an xmlrpc.Query.
func (ps ParamsetDescription) ReadFrom(q *xmlrpc.Query) {
	for n, v := range q.Map() {
		p := &ParameterDescription{}
		p.ReadFrom(v)
		if q.Err() != nil {
			break
		}
		ps[n] = p
	}
}

// ToValue returns an xmlrpc.Value for this paramset description.
func (ps ParamsetDescription) ToValue() (*xmlrpc.Value, error) {
	ms := make([]*xmlrpc.Member, len(ps))
	i := 0
	for n, p := range ps {
		v, err := p.ToValue()
		if err != nil {
			return nil, err
		}
		ms[i] = &xmlrpc.Member{Name: n, Value: v}
		i++
	}
	return &xmlrpc.Value{Struct: &xmlrpc.Struct{Members: ms}}, nil
}

// SplitAddress splits a full address into device and channel part.
func SplitAddress(address string) (deviceAddress string, channelAddress string) {
	if p := strings.IndexRune(address, ':'); p == -1 {
		deviceAddress = address
	} else {
		deviceAddress = address[0:p]
		channelAddress = address[p+1:]
	}
	return
}
