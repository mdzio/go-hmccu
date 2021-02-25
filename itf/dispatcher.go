package itf

import (
	"fmt"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

var svrLog = logging.Get("itf-server")

// A LogicLayer handles notifications from interface processes (of the CCU).
type LogicLayer interface {
	// A value has changed.
	Event(interfaceID, address, valueKey string, value interface{}) error

	// Devices are added.
	NewDevices(interfaceID string, devDescriptions []*DeviceDescription) error

	// Devices are deleted.
	DeleteDevices(interfaceID string, addresses []string) error

	// A device or channels has changed. hint=0: any changes; hint=1: number of
	// links changed
	UpdateDevice(interfaceID, address string, hint int) error

	// A device was replaced.
	ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress string) error

	// ReaddedDevice is called, when an already connected device is paired again
	// with the CCU. Deleted logical devices are listed in deletedAddresses.
	ReaddedDevice(interfaceID string, deletedAddresses []string) error

	// ListDevices is not forwarded.
}

// Dispatcher is an extended xmlrpc.Dispatcher for HM.
type Dispatcher struct {
	xmlrpc.BasicDispatcher
}

// NewDispatcher creates a new Dispatcher with HM specific RPC functions.
func NewDispatcher() *Dispatcher {
	d := &Dispatcher{}
	d.AddSystemMethods()
	return d
}

// AddLogicLayer adds handlers for a logic layer.
// After calling init on BidCos-RF normally following callbacks happen:
// system.listMethods, listDevices, newDevices and system.multicall with
// event's. Attention: listDevices is not forwarded to the receiver and returns
// always an empty device list to the interface process.
func (d *Dispatcher) AddLogicLayer(ll LogicLayer) {
	d.HandleFunc("event", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 4 {
			return nil, fmt.Errorf("Expected 4 arguments for event method: %d", len(q.Slice()))
		}
		interfaceID := q.Idx(0).String()
		address := q.Idx(1).String()
		valueKey := q.Idx(2).String()
		value := q.Idx(3).Any()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument for event method: %v", q.Err())
		}
		svrLog.Debugf("Call of method event received: %s, %s, %s, %v", interfaceID, address, valueKey, value)
		err := ll.Event(interfaceID, address, valueKey, value)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{FlatString: ""}, nil
	})

	// This method returns all the devices known to the logic layer for the
	// interface process with the the ID interface_id in the form of device
	// descriptions. This allows the interface process to perform a comparison by
	// calling newDevices() and deleteDevices(). For this to work, the logic layer
	// must remember this information at least partially. It is sufficient if the
	// ADDRESS and VERSION members of a device description are set.
	// Attention: This implementation returns always an empty device list.
	d.HandleFunc("listDevices", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 1 {
			return nil, fmt.Errorf("Expected one argument for listDevices method: %d", len(q.Slice()))
		}
		interfaceID := q.Idx(0).String()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument for listDevices method: %v", q.Err())
		}
		svrLog.Debugf("Call of method listDevices received: %s", interfaceID)
		return &xmlrpc.Value{Array: &xmlrpc.Array{Data: []*xmlrpc.Value{}}}, nil
	})

	d.HandleFunc("newDevices", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 2 {
			return nil, fmt.Errorf("Expected 2 arguments for newDevices method: %d", len(q.Slice()))
		}
		interfaceID := q.Idx(0).String()
		devDescriptions := q.Idx(1).Slice()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument for newDevices method: %v", q.Err())
		}
		svrLog.Debugf("Call of method newDevices received: %s, %d devices", interfaceID, len(devDescriptions))
		var descr []*DeviceDescription
		for _, q := range devDescriptions {
			d := &DeviceDescription{}
			d.ReadFrom(q)
			if q.Err() != nil {
				return nil, fmt.Errorf("Invalid device description for newDevices method: %v", q.Err())
			}
			descr = append(descr, d)
		}
		err := ll.NewDevices(interfaceID, descr)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{FlatString: ""}, nil
	})

	d.HandleFunc("deleteDevices", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 2 {
			return nil, fmt.Errorf("Expected 2 arguments for deleteDevices method: %d", len(q.Slice()))
		}
		interfaceID := q.Idx(0).String()
		addressesValue := q.Idx(1).Slice()
		var addresses []string
		for _, addrValue := range addressesValue {
			addresses = append(addresses, addrValue.String())
		}
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for deleteDevices method: %v", q.Err())
		}
		svrLog.Debugf("Call of method deleteDevices received: %s, %v", interfaceID, addresses)
		err := ll.DeleteDevices(interfaceID, addresses)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{FlatString: ""}, nil
	})

	d.HandleFunc("updateDevice", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 3 {
			return nil, fmt.Errorf("Expected 3 arguments for updateDevice method: %d", len(q.Slice()))
		}
		interfaceID := q.Idx(0).String()
		address := q.Idx(1).String()
		hint := q.Idx(2).Int()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for updateDevice method: %v", q.Err())
		}
		svrLog.Debugf("Call of method updateDevice received: %s, %s, %d", interfaceID, address, hint)
		err := ll.UpdateDevice(interfaceID, address, hint)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{FlatString: ""}, nil
	})

	d.HandleFunc("replaceDevice", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 3 {
			return nil, fmt.Errorf("Expected 3 arguments for replaceDevice method: %d", len(q.Slice()))
		}
		interfaceID := q.Idx(0).String()
		oldDeviceAddress := q.Idx(1).String()
		newDeviceAddress := q.Idx(2).String()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for replaceDevice method: %v", q.Err())
		}
		svrLog.Debugf("Call of method replaceDevice received: %s, %s, %s", interfaceID, oldDeviceAddress, newDeviceAddress)
		err := ll.ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{FlatString: ""}, nil
	})

	d.HandleFunc("readdedDevice", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 2 {
			return nil, fmt.Errorf("Expected 2 arguments for readdedDevice method: %d", len(q.Slice()))
		}
		interfaceID := q.Idx(0).String()
		deletedAddresses := q.Idx(1).Slice()
		var addresses []string
		for _, addrValue := range deletedAddresses {
			addresses = append(addresses, addrValue.String())
		}
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for readdedDevice method: %v", q.Err())
		}
		svrLog.Debugf("Call of method readdedDevice received: %s, %v", interfaceID, addresses)
		err := ll.ReaddedDevice(interfaceID, addresses)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{FlatString: ""}, nil
	})
}
