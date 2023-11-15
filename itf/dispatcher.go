package itf

import (
	"fmt"
	"strings"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-logging"
)

var svrLog = logging.Get("itf-server")

// A LogicLayer handles notifications from a device interface processes (of the CCU).
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

	// ListDevices is not forwarded. An empty list is always returned.
}

// A DeviceLayer is the API of a device interface process.
type DeviceLayer interface {
	// Init registers a new logic layer. The receiverAddress should have the
	// format protocol://hostname[:port][/Path]. If the second parameter is
	// omitted, the call is redirected to Deinit.
	Init(receiverAddress, interfaceID string) error

	// Deinit unregisters a logic layer. Init redirects the call to this
	// function, if the second parameter is ommited. This method is not exposed
	// over XML-RPC.
	Deinit(receiverAddress string) error

	// This method returns all devices known to the interface process in the
	// form of device descriptions.
	ListDevices() ([]*DeviceDescription, error)

	// This method deletes a device from the interface process.
	//
	// Flags:
	//   0x01=DELETE_FLAG_RESET
	//   0x02=DELETE_FLAG_FORCE
	//   0x04=DELETE_FLAG_DEFER
	DeleteDevice(deviceAddress string, flags int) error

	// This method returns the device description of the device passed as
	// deviceAddress.
	GetDeviceDescription(deviceAddress string) (*DeviceDescription, error)

	// This method is used to determine the description of a parameter set. The
	// parameter deviceAddress is the address of a logical device (e.g. returned
	// by ListDevices). The parameter paramsetType is "MASTER", "VALUES" or
	// "LINK".
	GetParamsetDescription(deviceAddress, paramsetType string) (ParamsetDescription, error)

	// This method reads a complete parameter set for a logical device. The
	// parameter deviceAddress is the address of a logical device. The parameter
	// paramsetKey is "MASTER" or "VALUES".
	GetParamset(deviceAddress string, paramsetKey string) (map[string]interface{}, error)

	// This method is used to write a complete parameter set for a logical
	// device. The parameter address is the address of a logical device. The
	// parameter paramsetKey is "MASTER" or "VALUES". Members not present in
	// values are simply not written and keep their old value.
	PutParamset(deviceAddress string, paramsetType string, paramset map[string]interface{}) error

	// This method is used to write a single value from the "VALUES" parameter
	// set. The parameter deviceAddress is the address of a logical device. The
	// parameter valueName is the name of the value to be written. The possible
	// values for valueName are taken from the ParamsetDescription of the
	// corresponding parameter set "VALUES". The value parameter is the value to
	// be written.
	SetValue(deviceAddress string, valueName string, value interface{}) error

	// This method reads a single value from the "VALUES" parameter set. The
	// parameter deviceAddress is the address of a logical device. The parameter
	// valueName is the name of the value to be read. The possible values for
	// valueName are derived from the ParamsetDescription of the corresponding
	// parameter set "VALUES".
	GetValue(deviceAddress string, valueName string) (interface{}, error)

	// When calling this function an event (called PONG in the following) is
	// generated and sent to all registered logic layers. Since the PONG event
	// is sent to all registered all registered logic layers (as with all other
	// events), one logic layer must expect to receive a logic layer must expect
	// to receive a PONG event without having previously called ping beforehand.
	//
	// The parameter callerId must be passed by the caller and is used as the
	// value of the PONG event. The content of the string is irrelevant. If no
	// exception occurs during processing, the method returns true is returned.
	//
	// The PONG event is delivered via the event method of the logic layer. The
	// address is CENTRAL", the key is "PONG" and the value is the callerId
	// passed in the ping call. passed in the ping call.
	Ping(callerID string) (bool, error)
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
		return &xmlrpc.Value{}, nil
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
		var descr []*DeviceDescription
		for _, q := range devDescriptions {
			d := &DeviceDescription{}
			d.ReadFrom(q)
			if q.Err() != nil {
				return nil, fmt.Errorf("Invalid device description for newDevices method: %v", q.Err())
			}
			descr = append(descr, d)
		}
		if svrLog.DebugEnabled() {
			var addrs []string
			for _, dd := range descr {
				addrs = append(addrs, dd.Address)
			}
			svrLog.Debugf("Call of method newDevices received: %s, %s", interfaceID, strings.Join(addrs, " "))
		}
		err := ll.NewDevices(interfaceID, descr)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{}, nil
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
		svrLog.Debugf("Call of method deleteDevices received: %s, %s", interfaceID, strings.Join(addresses, " "))
		err := ll.DeleteDevices(interfaceID, addresses)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{}, nil
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
		return &xmlrpc.Value{}, nil
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
		return &xmlrpc.Value{}, nil
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
		svrLog.Debugf("Call of method readdedDevice received: %s, %v", interfaceID, strings.Join(addresses, " "))
		err := ll.ReaddedDevice(interfaceID, addresses)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{}, nil
	})

	// XML-RPC: ? setReadyConfig(?)
	//
	// Attention: This call is not forwarded to LogicLayer.
	d.HandleFunc("setReadyConfig", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		svrLog.Debugf("Call of method setReadyConfig received, arguments: %s", args)
		// not needed, not implemented
		// return always an empty string
		return &xmlrpc.Value{}, nil
	})
}

// AddDeviceLayer adds handlers for a device layer.
func (d *Dispatcher) AddDeviceLayer(dl DeviceLayer) {

	// XML-RPC: void init(String url, String interface_id)
	d.HandleFunc("init", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		n := len(q.Slice())
		if n < 1 || n > 2 {
			return nil, fmt.Errorf("Expected 1 or 2 arguments for init method: %d", len(q.Slice()))
		}
		receiverAddress := q.Idx(0).String()
		var interfaceID string
		if n == 2 {
			interfaceID = q.Idx(1).String()
		}
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for init method: %v", q.Err())
		}
		svrLog.Debugf("Call of method init received: %s, %s", receiverAddress, interfaceID)
		var err error
		if n == 2 {
			err = dl.Init(receiverAddress, interfaceID)
		} else {
			err = dl.Deinit(receiverAddress)
		}
		if err != nil {
			return nil, err
		}
		// on success return empty response
		return &xmlrpc.Value{}, nil
	})

	// XML-RPC: Array<DeviceDescription> listDevices()
	d.HandleFunc("listDevices", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		// ignore arguments
		// get device list
		dds, err := dl.ListDevices()
		if err != nil {
			return nil, err
		}
		// build XML-RPC array
		arr := make([]*xmlrpc.Value, len(dds))
		for idx := range dds {
			arr[idx] = dds[idx].ToValue()
		}
		return &xmlrpc.Value{Array: &xmlrpc.Array{Data: arr}}, nil
	})

	// XML-RPC: void deleteDevice(String address, Integer flags)
	d.HandleFunc("deleteDevice", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 2 {
			return nil, fmt.Errorf("Expected 2 arguments for deleteDevice method: %d", len(q.Slice()))
		}
		address := q.Idx(0).String()
		flags := q.Idx(1).Int()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for deleteDevice method: %v", q.Err())
		}
		svrLog.Debugf("Call of method deleteDevice received: %s, %d", address, flags)
		err := dl.DeleteDevice(address, flags)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{}, nil
	})

	// XML-RPC: DeviceDescription getDeviceDescription(String address)
	d.HandleFunc("getDeviceDescription", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 1 {
			return nil, fmt.Errorf("Expected 1 argument for getDeviceDescription method: %d", len(q.Slice()))
		}
		address := q.Idx(0).String()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for getDeviceDescription method: %v", q.Err())
		}
		svrLog.Debugf("Call of method getDeviceDescription received: %s", address)
		descr, err := dl.GetDeviceDescription(address)
		if err != nil {
			return nil, err
		}
		return descr.ToValue(), nil
	})

	// XML-RPC: ParamsetDescription getParamsetDescription(String address, String paramset_type)
	d.HandleFunc("getParamsetDescription", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 2 {
			return nil, fmt.Errorf("Expected 2 arguments for getParamsetDescription method: %d", len(q.Slice()))
		}
		deviceAddress := q.Idx(0).String()
		paramsetType := q.Idx(1).String()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for getParamsetDescription method: %v", q.Err())
		}
		svrLog.Debugf("Call of method getParamsetDescription received: %s, %s", deviceAddress, paramsetType)
		psd, err := dl.GetParamsetDescription(deviceAddress, paramsetType)
		if err != nil {
			return nil, err
		}
		psdv, err := psd.ToValue()
		if err != nil {
			return nil, fmt.Errorf("Conversion to XML-RPC value failed: %v", err)
		}
		return psdv, nil
	})

	// XML-RPC: Paramset getParamset(String address, String paramset_key)
	d.HandleFunc("getParamset", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 2 {
			return nil, fmt.Errorf("Expected 2 arguments for getParamset method: %d", len(q.Slice()))
		}
		deviceAddress := q.Idx(0).String()
		paramsetKey := q.Idx(1).String()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for getParamset method: %v", q.Err())
		}
		svrLog.Debugf("Call of method getParamset received: %s, %s", deviceAddress, paramsetKey)
		ps, err := dl.GetParamset(deviceAddress, paramsetKey)
		if err != nil {
			return nil, err
		}
		psv, err := xmlrpc.NewValue(ps)
		if err != nil {
			return nil, fmt.Errorf("Conversion to XML-RPC value failed: %v", err)
		}
		return psv, nil
	})

	// XML-RPC: void putParamset(String address, String paramset_key, Paramset set)
	d.HandleFunc("putParamset", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 3 {
			return nil, fmt.Errorf("Expected 3 arguments for putParamset method: %d", len(q.Slice()))
		}
		deviceAddress := q.Idx(0).String()
		paramsetKey := q.Idx(1).String()
		paramset := q.Idx(2).Map()
		// convert values
		ps := make(map[string]interface{})
		for n, v := range paramset {
			ps[n] = v.Any()
		}
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for putParamset method: %v", q.Err())
		}
		svrLog.Debugf("Call of method putParamset received: %s, %s", deviceAddress, paramsetKey)
		err := dl.PutParamset(deviceAddress, paramsetKey, ps)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{}, nil
	})

	// XML-RPC: ValueType getValue(String address, String value_key)
	d.HandleFunc("getValue", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 2 {
			return nil, fmt.Errorf("Expected 2 arguments for getValue method: %d", len(q.Slice()))
		}
		deviceAddress := q.Idx(0).String()
		valueKey := q.Idx(1).String()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for getValue method: %v", q.Err())
		}
		value, err := dl.GetValue(deviceAddress, valueKey)
		if err != nil {
			return nil, err
		}
		v, err := xmlrpc.NewValue(value)
		if err != nil {
			return nil, fmt.Errorf("Conversion to XML-RPC value failed: %v", err)
		}
		return v, nil
	})

	// XML-RPC: void setValue(String address, String value_key, ValueType value)
	d.HandleFunc("setValue", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 3 {
			return nil, fmt.Errorf("Expected 3 arguments for setValue method: %d", len(q.Slice()))
		}
		deviceAddress := q.Idx(0).String()
		valueKey := q.Idx(1).String()
		value := q.Idx(2).Any()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for setValue method: %v", q.Err())
		}
		err := dl.SetValue(deviceAddress, valueKey, value)
		if err != nil {
			return nil, err
		}
		return &xmlrpc.Value{}, nil
	})

	// XML-RPC: bool ping(String callerId)
	d.HandleFunc("ping", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		q := xmlrpc.Q(args)
		if len(q.Slice()) != 1 {
			return nil, fmt.Errorf("Expected 1 argument for ping method: %d", len(q.Slice()))
		}
		callerID := q.Idx(0).String()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument(s) for ping method: %v", q.Err())
		}
		res, err := dl.Ping(callerID)
		if err != nil {
			return nil, err
		}
		return xmlrpc.NewBool(res), nil
	})

	// XML-RPC: Boolean reportValueUsage(String address, String value_id,
	// Integer ref_counter)
	//
	// Attention: This call is not forwarded to DeviceLayer.
	d.HandleFunc("reportValueUsage", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		svrLog.Debugf("Call of method reportValueUsage received, arguments: %s", args)
		// not needed, not implemented
		// return always true: action succeeded
		return &xmlrpc.Value{Boolean: "1"}, nil
	})

	// XML-RPC: Array<Struct>getLinks(String address, Integer flags)
	//
	// Attention: This call is not forwarded to DeviceLayer.
	d.HandleFunc("getLinks", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		svrLog.Debugf("Call of method getLinks received, arguments: %s", args)
		// not needed, not implemented
		// return always an empty array
		return &xmlrpc.Value{Array: &xmlrpc.Array{}}, nil
	})

	// XML-RPC: String getParamsetId(String address, String type)
	//
	// Attention: This call is not forwarded to DeviceLayer.
	d.HandleFunc("getParamsetId", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		svrLog.Debugf("Call of method getParamsetId received, arguments: %s", args)
		// not needed, not implemented
		// return always an empty string
		return &xmlrpc.Value{}, nil
	})

	// XML-RPC: ? firmwareUpdateStatusChanged(?)
	//
	// Attention: This call is not forwarded to DeviceLayer.
	d.HandleFunc("firmwareUpdateStatusChanged", func(args *xmlrpc.Value) (*xmlrpc.Value, error) {
		svrLog.Debugf("Call of method firmwareUpdateStatusChanged received, arguments: %s", args)
		// not needed, not implemented
		// return always an empty string
		return &xmlrpc.Value{}, nil
	})
}
