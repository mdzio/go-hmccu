package itf

import (
	"fmt"
	"github.com/mdzio/go-hmccu/binrpc"
	"github.com/mdzio/go-hmccu/model"

	"github.com/mdzio/go-hmccu/xmlrpc"
	"github.com/mdzio/go-logging"
)

var svrLog = logging.Get("itf-server")

// A Receiver gets all notifications from the CCU.
type Receiver interface {
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
}

// Handler forwards HM XML-RPC interface calls to the receiver. After calling
// init on BidCos-RF normally following callbacks happen: system.listMethods,
// listDevices, newDevices and system.multicall with event's. Attention:
// listDevices is not forwarded to the receiver and returns always an empty
// device list to the CCU.
type Handler struct {
	xmlrpc.Handler
	receiver Receiver
}

// NewHandler creates a new HM XML-RPC handler.
func NewHandler(receiver Receiver) *Handler {
	h := &Handler{
		receiver: receiver,
	}
	h.SystemMethods()

	h.HandleFunc("event", eventHandleFunc(h.receiver))

	// attention: this implementation returns always an empty device list.
	h.HandleFunc("listDevices", listDevicesHandleFunc())

	h.HandleFunc("newDevices", newDevicesHandleFunc(h.receiver))

	h.HandleFunc("deleteDevices", deleteDevicesHandleFunc(h.receiver))

	h.HandleFunc("updateDevice", updateDevicesHandleFunc(h.receiver))

	h.HandleFunc("replaceDevice", replaceDevicesHandleFunc(h.receiver))

	h.HandleFunc("readdedDevice", readdedDevicesHandleFunc(h.receiver))

	return h
}

// BinRpcHandler forwards CUxD BIN-RPC interface calls to the receiver.
// CUxD does not call any method after init has been called. Therefore
// only event callbacks are configured.
type BinRpcHandler struct {
	binrpc.Handler
	receiver Receiver
}

// NewRpcHandler creates a new CUxD Bin-RPC handler.
func NewRpcHandler(receiver Receiver) *BinRpcHandler {
	h := &BinRpcHandler{
		receiver: receiver,
	}
	h.SystemMethods()

	h.HandleFunc("event", eventHandleFunc(h.receiver))

	return h
}

func eventHandleFunc(r Receiver) func(args *model.Value) (*model.Value, error) {
	return func(args *model.Value) (*model.Value, error) {
		q := model.Q(args)
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
		err := r.Event(interfaceID, address, valueKey, value)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	}
}

func listDevicesHandleFunc() func(args *model.Value) (*model.Value, error) {
	return func(args *model.Value) (*model.Value, error) {
		q := model.Q(args)
		if len(q.Slice()) != 1 {
			return nil, fmt.Errorf("Expected one argument for listDevices method: %d", len(q.Slice()))
		}
		interfaceID := q.Idx(0).String()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument for listDevices method: %v", q.Err())
		}
		svrLog.Debugf("Call of method listDevices received: %s", interfaceID)
		return &model.Value{Array: &model.Array{Data: []*model.Value{}}}, nil
	}
}

func newDevicesHandleFunc(r Receiver) func(args *model.Value) (*model.Value, error) {
	return func(args *model.Value) (*model.Value, error) {
		q := model.Q(args)
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
		err := r.NewDevices(interfaceID, descr)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	}
}

func deleteDevicesHandleFunc(r Receiver) func(args *model.Value) (*model.Value, error) {
	return func(args *model.Value) (*model.Value, error) {
		q := model.Q(args)
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
		err := r.DeleteDevices(interfaceID, addresses)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	}
}

func updateDevicesHandleFunc(r Receiver) func(args *model.Value) (*model.Value, error) {
	return func(args *model.Value) (*model.Value, error) {
		q := model.Q(args)
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
		err := r.UpdateDevice(interfaceID, address, hint)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	}
}

func replaceDevicesHandleFunc(r Receiver) func(args *model.Value) (*model.Value, error) {
	return func(args *model.Value) (*model.Value, error) {
		q := model.Q(args)
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
		err := r.ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	}
}

func readdedDevicesHandleFunc(r Receiver) func(args *model.Value) (*model.Value, error) {
	return func(args *model.Value) (*model.Value, error) {
		q := model.Q(args)
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
		err := r.ReaddedDevice(interfaceID, addresses)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	}
}
