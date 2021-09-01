package vdevices

import (
	"github.com/mdzio/go-hmccu/itf"
)

// addInstallTest adds the INSTALL_TEST parameter for simulating a channel/device test
func addInstallTest(ch *Channel) {
	p := NewBoolParameter("INSTALL_TEST")
	p.description.Type = itf.ParameterTypeAction
	p.description.Operations = itf.ParameterOperationWrite
	p.description.Flags = itf.ParameterFlagVisible | itf.ParameterFlagInternal
	ch.AddValueParam(&p.Parameter)
}

// MaintenanceChannel is a standard HM device maintenance channel. The first
// channel (Index: 0) of every HM device should be a maintenance channel.
type MaintenanceChannel struct {
	Channel

	unreach       *BoolParameter
	stickyUnreach *BoolParameter
}

// NewMaintenanceChannel creates a new maintenance channel and adds it to the
// device.
func NewMaintenanceChannel(device *Device) *MaintenanceChannel {
	c := new(MaintenanceChannel)
	c.Channel.Init("MAINTENANCE")
	c.description.Flags = itf.DeviceFlagVisible | itf.DeviceFlagInternal
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add UNREACH parameter
	c.unreach = NewBoolParameter("UNREACH")
	c.unreach.description.Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	c.unreach.description.Flags = itf.ParameterFlagVisible | itf.ParameterFlagService
	c.AddValueParam(&c.unreach.Parameter)

	// add STICKY_UNREACH parameter
	c.stickyUnreach = NewBoolParameter("STICKY_UNREACH")
	c.stickyUnreach.description.Operations = itf.ParameterOperationRead | itf.ParameterOperationWrite | itf.ParameterOperationEvent
	c.stickyUnreach.description.Flags = itf.ParameterFlagVisible | itf.ParameterFlagService | itf.ParameterFlagSticky
	c.AddValueParam(&c.stickyUnreach.Parameter)
	return c
}

// SetUnreach sets the connection state of the device.
func (c *MaintenanceChannel) SetUnreach(value bool) {
	c.unreach.SetValueUnchecked(value)
	if value {
		c.stickyUnreach.SetValueUnchecked(true)
	}
}

// SwitchChannel implements a standard HM switch channel.
type SwitchChannel struct {
	Channel

	// This callback is executed when an external system wants to change the
	// state. Only if this function returns true, the state is actually set.
	OnSetState func(value bool) (ok bool)

	state *BoolParameter
}

// NewSwitchChannel creates a new HM switch channel and adds it to the device.
// The field OnSetState must be set to be able to react to external value
// changes.
func NewSwitchChannel(device *Device) *SwitchChannel {
	c := new(SwitchChannel)
	c.Channel.Init("SWITCH")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add STATE parameter
	c.state = NewBoolParameter("STATE")
	c.state.description.Control = "SWITCH.STATE"
	c.state.OnSetValue = func(value bool) bool {
		if c.OnSetState != nil {
			return c.OnSetState(value)
		} else {
			return true
		}
	}
	c.AddValueParam(&c.state.Parameter)
	return c
}

// SetState sets the state of the switch.
func (c *SwitchChannel) SetState(value bool) {
	c.state.SetValueUnchecked(value)
}

// State returns the state of the switch.
func (c *SwitchChannel) State() bool {
	return c.state.Value().(bool)
}

// KeyChannel implements a standard HM key channel.
type KeyChannel struct {
	Channel
	OnPressShort func() bool
	OnPressLong  func() bool

	pressShort *BoolParameter
	pressLong  *BoolParameter
}

// NewKeyChannel creates a new HM key channel and adds it to the device.
func NewKeyChannel(device *Device) *KeyChannel {
	c := new(KeyChannel)
	c.Channel.Init("KEY_TRANSCEIVER")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add PRESS_SHORT parameter
	c.pressShort = NewBoolParameter("PRESS_SHORT")
	c.pressShort.description.Type = itf.ParameterTypeAction
	c.pressShort.description.Operations = itf.ParameterOperationWrite | itf.ParameterOperationEvent
	c.pressShort.description.Control = "BUTTON.SHORT"
	c.pressShort.OnSetValue = func(value bool) bool {
		if c.OnPressShort != nil {
			return c.OnPressShort()
		} else {
			return true
		}
	}
	c.AddValueParam(&c.pressShort.Parameter)

	// add PRESS_LONG parameter
	c.pressLong = NewBoolParameter("PRESS_LONG")
	c.pressLong.description.Type = itf.ParameterTypeAction
	c.pressLong.description.Operations = itf.ParameterOperationWrite | itf.ParameterOperationEvent
	c.pressLong.description.Control = "BUTTON.LONG"
	c.pressLong.OnSetValue = func(value bool) bool {
		if c.OnPressLong != nil {
			return c.OnPressLong()
		} else {
			return true
		}
	}
	c.AddValueParam(&c.pressLong.Parameter)
	return c
}

// PressShort sends a press short event.
func (c *KeyChannel) PressShort() {
	c.pressShort.SetValueUnchecked(true)
}

// PressShort sends a press long event.
func (c *KeyChannel) PressLong() {
	c.pressShort.SetValueUnchecked(true)
}

// AnalogInputChannel implements a HM analog input channel (e.g.
// HmIP-MIO16-PCB:1).
type AnalogInputChannel struct {
	Channel

	// These callbacks are executed when an external system wants to change the
	// values. Only if the function returns true, the value is actually set.
	OnSetVoltage       func(value float64) (ok bool)
	OnSetVoltageStatus func(value int) (ok bool)

	voltage       *FloatParameter
	voltageStatus *IntParameter
}

// NewAnalogInputChannel creates a new HM analog input channel and adds it to the device.
// The field OnSetVoltage must be set to be able to react to external value
// changes.
func NewAnalogInputChannel(device *Device) *AnalogInputChannel {
	c := new(AnalogInputChannel)
	c.Channel.Init("ANALOG_INPUT_TRANSMITTER")
	// adding channel to device also initializes some fields
	device.AddChannel(&c.Channel)
	addInstallTest(&c.Channel)

	// add VOLTAGE parameter
	c.voltage = NewFloatParameter("VOLTAGE")
	c.voltage.description.Control = "ANALOG_INPUT.VOLTAGE"
	c.voltage.OnSetValue = func(value float64) bool {
		if c.OnSetVoltage != nil {
			return c.OnSetVoltage(value)
		} else {
			return true
		}
	}
	c.AddValueParam(&c.voltage.Parameter)

	// add VOLTAGE_STATUS parameter
	c.voltageStatus = NewIntParameter("VOLTAGE_STATUS")
	c.voltageStatus.description.Type = itf.ParameterTypeEnum
	c.voltageStatus.description.Control = "ANALOG_INPUT.VOLTAGE_STATUS"

	// Following values are reported by an analog input of a HmIP-MIO16-PCB:1.
	// c.voltageStatus.description.Default = "NORMAL"
	// c.voltageStatus.description.Max = "OVERFLOW"
	// c.voltageStatus.description.Min = "NORMAL"
	// c.voltageStatus.description.ValueList = []string{"NORMAL", "UNKNOWN", "OVERFLOW"}
	// Even when using these values, the VOLTAGE_STATUS is not displayed in the
	// Web-UI of the CCU, e.g. as a program trigger, as it is for a real device.
	// Maybe someone can explain. With the following settings at least all
	// possible values of the ENUM are displayed, although the value 1 is
	// normally hidden.

	c.voltageStatus.description.Default = 0
	c.voltageStatus.description.Max = 3
	c.voltageStatus.description.Min = 0
	c.voltageStatus.description.ValueList = []string{"NORMAL", "UNKNOWN", "OVERFLOW", "UNDERFLOW"}

	c.voltageStatus.OnSetValue = func(value int) bool {
		if c.OnSetVoltage != nil {
			return c.OnSetVoltageStatus(value)
		} else {
			return true
		}
	}
	c.AddValueParam(&c.voltageStatus.Parameter)
	return c
}

// SetVoltage sets the voltage of the analog input.
func (c *AnalogInputChannel) SetVoltage(value float64) {
	c.voltage.SetValueUnchecked(value)
}

// Voltage returns the voltage of the analog input.
func (c *AnalogInputChannel) Voltage() float64 {
	return c.voltage.Value().(float64)
}

// SetVoltageStatus sets the voltage status of the analog input.
func (c *AnalogInputChannel) SetVoltageStatus(value int) {
	c.voltageStatus.SetValueUnchecked(value)
}

// VoltageStatus returns the voltage status of the analog input.
func (c *AnalogInputChannel) VoltageStatus() int {
	return c.voltageStatus.Value().(int)
}
