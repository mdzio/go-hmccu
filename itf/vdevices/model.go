package vdevices

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/mdzio/go-hmccu/itf"
)

// Device is a generic container for channels and device master parameters. It
// implements interface GenericDevice. The structure of a device (channels and
// parameters) must not be changed after adding to the Container.
type Device struct {
	description    *itf.DeviceDescription
	masterParamset Paramset
	channels       []*Channel
	locker         sync.Mutex
	publisher      EventPublisher

	// Handler for dispose of device (optional)
	OnDispose func()
}

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
		channels:  make([]*Channel, 0),
	}
}

// Description implements interface GenericDevice.
func (d *Device) Description() *itf.DeviceDescription {
	return d.description
}

// Channels implements interface GenericDevice.
func (d *Device) Channels() []GenericChannel {
	gc := make([]GenericChannel, len(d.channels))
	for idx := range d.channels {
		gc[idx] = d.channels[idx]
	}
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
func (d *Device) AddChannel(channel *Channel) {
	// complement channel description
	idx := len(d.channels)
	descr := channel.Description()
	descr.Parent = d.description.Address
	descr.ParentType = d.description.Type
	descr.Address = d.description.Address + ":" + strconv.Itoa(idx)
	descr.Index = idx
	// add channel to device
	channel.publisher = d.publisher
	d.channels = append(d.channels, channel)
	d.description.Children = append(d.description.Children, descr.Address)
}

// AddMasterParam adds a parameter to the master paramset.
func (d *Device) AddMasterParam(parameter *Parameter) {
	parameter.parentDescr = d.description
	parameter.locker = &d.locker
	d.masterParamset.Add(parameter)
}

// Locker returns the device locker.
func (d *Device) Locker() sync.Locker {
	return &d.locker
}

// Dispose must be called, when the device should free resources. Function
// OnDispose gets called, if specified. Afterwards Dispose of each channel is
// invoked.
func (d *Device) Dispose() {
	if d.OnDispose != nil {
		d.OnDispose()
	}
	// dispose channels
	for _, ch := range d.channels {
		ch.Dispose()
	}
}

// Channel implements interface GenericChannel.
type Channel struct {
	description    *itf.DeviceDescription
	masterParamset Paramset
	valueParamset  Paramset
	locker         sync.Mutex
	publisher      EventPublisher

	// Handler for dispose of channel (optional)
	OnDispose func()
}

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

// AddMasterParam adds a parameter to the MASTER paramset.
func (c *Channel) AddMasterParam(parameter *Parameter) {
	parameter.parentDescr = c.description
	parameter.locker = &c.locker
	c.masterParamset.Add(parameter)
}

// AddValueParam adds a parameter to the VALUES paramset.
func (c *Channel) AddValueParam(parameter *Parameter) {
	parameter.parentDescr = c.description
	parameter.locker = &c.locker
	parameter.publisher = c.publisher
	c.valueParamset.Add(parameter)
}

// Locker returns the channel locker.
func (c *Channel) Locker() sync.Locker {
	return &c.locker
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
	params map[string]*Parameter
}

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

// Add adds a parameter to this parameter set.
func (s *Paramset) Add(param *Parameter) {
	if s.params == nil {
		s.params = make(map[string]*Parameter)
	}
	s.params[param.description.ID] = param
}
