package vdevices

import (
	"fmt"
	"strconv"

	"github.com/mdzio/go-hmccu/itf"
)

// Device is a generic container for channels and device master parameters. It
// implements interface GenericDevice. The structure of a device (channels and
// parameters) must not be changed after adding to the Container.
type Device struct {
	description    *itf.DeviceDescription
	masterParamset Paramset
	channels       []GenericChannel
	publisher      EventPublisher

	// Handler for dispose of device (optional)
	OnDispose func()
}

// check interface implementation
var _ GenericDevice = (*Device)(nil)

// NewDevice creates a Device.
func NewDevice(address, deviceType string, publisher EventPublisher) *Device {
	return &Device{
		description: &itf.DeviceDescription{
			Type:      deviceType,
			Address:   address,
			Paramsets: []string{"MASTER"},
			Flags:     itf.DeviceFlagVisible,
			Version:   1,
		},
		publisher: publisher,
		channels:  make([]GenericChannel, 0),
	}
}

// Description implements interface GenericDevice.
func (d *Device) Description() *itf.DeviceDescription {
	return d.description
}

// Channels implements interface GenericDevice.
func (d *Device) Channels() []GenericChannel {
	gc := make([]GenericChannel, len(d.channels))
	copy(gc, d.channels)
	return gc
}

// Channel implements interface GenericDevice.
func (d *Device) Channel(channelAddress string) (GenericChannel, error) {
	ch, err := strconv.Atoi(channelAddress)
	if err != nil || ch < 0 || ch >= len(d.channels) {
		return nil, fmt.Errorf("Channel in device %s not found: %s", d.description.Address, channelAddress)
	}
	return d.channels[ch], nil
}

// MasterParamset implements interface GenericDevice.
func (d *Device) MasterParamset() GenericParamset {
	return &d.masterParamset
}

// AddChannel binds a channel to the device. Following fields in the channels
// description are initialized: Parent, ParentType, Address, Index. Publisher of
// the channel is set to the publisher of the device.
func (d *Device) AddChannel(channel GenericChannel) {
	// complement channel description
	idx := len(d.channels)
	descr := channel.Description()
	descr.Parent = d.description.Address
	descr.ParentType = d.description.Type
	descr.Address = d.description.Address + ":" + strconv.Itoa(idx)
	descr.Index = idx
	// add channel to device
	channel.SetPublisher(d.publisher)
	d.channels = append(d.channels, channel)
	d.description.Children = append(d.description.Children, descr.Address)
}

// AddMasterParam adds a parameter to the master paramset.
func (d *Device) AddMasterParam(parameter GenericParameter) {
	parameter.SetParentDescr(d.description)
	d.masterParamset.Add(parameter)
}

// Dispose must be called, when the device should free resources. Function
// OnDispose gets called, if specified. Afterwards Dispose of each channel is
// invoked.
func (d *Device) Dispose() {
	// dispose channels
	for _, ch := range d.channels {
		ch.Dispose()
	}
	if d.OnDispose != nil {
		d.OnDispose()
	}
}

// Channel implements interface GenericChannel.
type Channel struct {
	description    *itf.DeviceDescription
	masterParamset Paramset
	valueParamset  Paramset
	publisher      EventPublisher

	// Handler for dispose of channel (optional)
	OnDispose func()
}

// check interface implementation
var _ GenericChannel = (*Channel)(nil)

// Init initializes Channel. This function must be called before any other
// member function.
func (c *Channel) Init(channelType string) {
	c.description = &itf.DeviceDescription{
		Type:      channelType,
		Paramsets: []string{"MASTER", "VALUES"},
		Flags:     itf.DeviceFlagVisible,
		Version:   1,
	}
}

// Description implements interface GenericChannel.
func (c *Channel) Description() *itf.DeviceDescription {
	return c.description
}

// MasterParamset implements interface GenericChannel.
func (c *Channel) MasterParamset() GenericParamset {
	return &c.masterParamset
}

// ValueParamset implements interface GenericChannel.
func (c *Channel) ValueParamset() GenericParamset {
	return &c.valueParamset
}

// SetPublisher implements interface GenericChannel.
func (c *Channel) SetPublisher(pub EventPublisher) {
	c.publisher = pub
}

// AddMasterParam adds a parameter to the MASTER paramset. OperationEvent is
// cleared. TabOrder is auto generated.
func (c *Channel) AddMasterParam(parameter GenericParameter) {
	parameter.SetParentDescr(c.description)
	// do not set a publisher, clear event operation
	parameter.Description().Operations = parameter.Description().Operations & ^itf.ParameterOperationEvent
	// auto generate tab order
	parameter.Description().TabOrder = c.masterParamset.Len()
	c.masterParamset.Add(parameter)
}

// AddValueParam adds a parameter to the VALUES paramset.
func (c *Channel) AddValueParam(parameter GenericParameter) {
	parameter.SetParentDescr(c.description)
	parameter.SetPublisher(c.publisher)
	c.valueParamset.Add(parameter)
}

// Dispose must be called, when the channel should free resources. Function
// OnDispose gets called, if specified.
func (c *Channel) Dispose() {
	if c.OnDispose != nil {
		c.OnDispose()
	}
}

// Paramset implements GenericParamset.
type Paramset struct {
	params map[string]GenericParameter

	// The optional putParamsetHandler is called after executing the RPC method
	// putParamset. The corresponding device or channel is locked while
	// executed.
	putParamsetHandler func()
}

// check interface implementation
var _ GenericParamset = (*Paramset)(nil)

// Parameters implements interface GenericParamset.
func (s *Paramset) Parameters() []GenericParameter {
	ps := make([]GenericParameter, 0, len(s.params))
	for _, p := range s.params {
		ps = append(ps, p)
	}
	return ps
}

// Parameter implements interface GenericParamset.
func (s *Paramset) Parameter(id string) (GenericParameter, error) {
	p, ok := s.params[id]
	if !ok {
		return nil, fmt.Errorf("Unknown parameter: %s", id)
	}
	return p, nil
}

// Len implements interface GenericParamset.
func (s *Paramset) Len() int {
	return len(s.params)
}

// NotifyPutParamset implements interface GenericParamset.
func (s *Paramset) NotifyPutParamset() {
	if s.putParamsetHandler != nil {
		s.putParamsetHandler()
	}
}

// HandlePutParamset implements interface GenericParamset.
func (s *Paramset) HandlePutParamset(f func()) {
	s.putParamsetHandler = f
}

// Add adds a parameter to this parameter set.
func (s *Paramset) Add(param GenericParameter) {
	if s.params == nil {
		s.params = make(map[string]GenericParameter)
	}
	s.params[param.Description().ID] = param
}
