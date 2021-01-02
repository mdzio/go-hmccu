package itf

import (
	"fmt"
	"github.com/mdzio/go-hmccu/binrpc"
	"github.com/mdzio/go-hmccu/model"

	"github.com/mdzio/go-logging"
)

var rpcsvrLog = logging.Get("itf-server")

// Handler forwards HM XML-RPC interface calls to the receiver. After calling
// init on BidCos-RF normally following callbacks happen: system.listMethods,
// listDevices, newDevices and system.multicall with event's. Attention:
// listDevices is not forwarded to the receiver and returns always an empty
// device list to the CCU.
type RpcHandler struct {
	binrpc.Handler
	receiver Receiver
}

// NewHandler creates a new HM XML-RPC handler.
func NewRpcHandler(receiver Receiver) *RpcHandler {
	h := &RpcHandler{
		receiver: receiver,
	}
	h.SystemMethods()

	h.HandleFunc("event", func(args *model.Value) (*model.Value, error) {
		q := model.Q(args)
		if len(q.Slice()) != 4 {
			return nil, fmt.Errorf("Expected 4 arguments for event method: %d", len(q.Slice()))
		}
		interfaceID := "CCU-Jack-" + q.Idx(0).String()
		address := q.Idx(1).String()
		valueKey := q.Idx(2).String()
		value := q.Idx(3).Any()
		if q.Err() != nil {
			return nil, fmt.Errorf("Invalid argument for event method: %v", q.Err())
		}
		svrLog.Debugf("Call of method event received: %s, %s, %s, %v", interfaceID, address, valueKey, value)
		err := h.receiver.Event(interfaceID, address, valueKey, value)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	})

	// attention: this implementation returns always an empty device list.
	h.HandleFunc("listDevices", func(args *model.Value) (*model.Value, error) {
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
	})

	h.HandleFunc("newDevices", func(args *model.Value) (*model.Value, error) {
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
		err := h.receiver.NewDevices(interfaceID, descr)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	})

	h.HandleFunc("deleteDevices", func(args *model.Value) (*model.Value, error) {
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
		err := h.receiver.DeleteDevices(interfaceID, addresses)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	})

	h.HandleFunc("updateDevice", func(args *model.Value) (*model.Value, error) {
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
		err := h.receiver.UpdateDevice(interfaceID, address, hint)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	})

	h.HandleFunc("replaceDevice", func(args *model.Value) (*model.Value, error) {
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
		err := h.receiver.ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	})

	h.HandleFunc("readdedDevice", func(args *model.Value) (*model.Value, error) {
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
		err := h.receiver.ReaddedDevice(interfaceID, addresses)
		if err != nil {
			return nil, err
		}
		return &model.Value{FlatString: ""}, nil
	})

	return h
}
