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
	device.bindChannel(&c.Channel)
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
	c.locker.Lock()
	defer c.locker.Unlock()
	c.unreach.RawSetValue(value)
	if value {
		c.stickyUnreach.RawSetValue(true)
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
	device.bindChannel(&c.Channel)
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
	c.locker.Lock()
	defer c.locker.Unlock()
	c.state.RawSetValue(value)
}

// State returns the state of the switch.
func (c *SwitchChannel) State() bool {
	c.locker.Lock()
	defer c.locker.Unlock()
	return c.state.RawValue()
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
	device.bindChannel(&c.Channel)
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
	c.locker.Lock()
	defer c.locker.Unlock()
	c.pressShort.RawSetValue(true)
}

// PressShort sends a press long event.
func (c *KeyChannel) PressLong() {
	c.locker.Lock()
	defer c.locker.Unlock()
	c.pressShort.RawSetValue(true)
}
