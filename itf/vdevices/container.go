package vdevices

import (
	"fmt"
	"sync"

	"github.com/mdzio/go-hmccu/itf"
)

// GenericDevice that can be used by Handler.
type GenericDevice interface {
	Description() *itf.DeviceDescription

	Channels() []GenericChannel
	Channel(channelAddress string) (GenericChannel, error)

	MasterParamset() GenericParamset

	Dispose()
}

// GenericChannel that can be used by Handler.
type GenericChannel interface {
	Description() *itf.DeviceDescription

	MasterParamset() GenericParamset
	ValueParamset() GenericParamset
}

// GenericParamset that can be used by Handler.
type GenericParamset interface {
	Parameters() []GenericParameter
	Parameter(id string) (GenericParameter, error)
}

// GenericParameter that can be used by Handler.
type GenericParameter interface {
	Description() *itf.ParameterDescription

	SetValue(value interface{}) error
	Value() interface{}
}

// A Container manages virtual devices and can be used by Handler. Devices can
// be added and removed at any time.
type Container struct {
	// Synchronizer updates the device lists in the logic layers.
	Synchronizer Synchronizer

	mtx     sync.RWMutex
	devices map[string]GenericDevice // key: address
}

// NewContainer creates a new device container.
func NewContainer() *Container {
	return &Container{
		devices: make(map[string]GenericDevice),
	}
}

// Dispose releases all devices and calls Dispose on them.
func (c *Container) Dispose() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	for _, d := range c.devices {
		d.Dispose()
	}
	c.devices = nil
}

// AddDevice adds the specified device to the container. The structure of a
// device, e.g. the channels and paramsets, must not change after adding the
// device.
func (c *Container) AddDevice(device GenericDevice) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	addr := device.Description().Address
	_, found := c.devices[addr]
	if found {
		return fmt.Errorf("Device already exists: %s", addr)
	}
	c.devices[addr] = device
	c.Synchronizer.Synchronize()
	return nil
}

// RemoveDevice removes the specified device from the container. If the device
// implements Disposer, Dispose gets called.
func (c *Container) RemoveDevice(address string) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	d, found := c.devices[address]
	if !found {
		return fmt.Errorf("Device not found: %s", address)
	}
	delete(c.devices, address)
	d.Dispose()
	c.Synchronizer.Synchronize()
	return nil
}

// Device returns the device for the address.
func (c *Container) Device(address string) (GenericDevice, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	d, found := c.devices[address]
	if !found {
		return nil, fmt.Errorf("Device not found: %s", address)
	}
	return d, nil
}

// Devices returns all devices.
func (c *Container) Devices() []GenericDevice {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	ds := make([]GenericDevice, 0, len(c.devices))
	for _, d := range c.devices {
		ds = append(ds, d)
	}
	return ds
}
