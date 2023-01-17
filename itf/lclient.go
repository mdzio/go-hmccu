package itf

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"

	"github.com/mdzio/go-logging"
)

var lclnLog = logging.Get("itf-l-client")

// LogicLayerClient provides access to the HomeMatic XML-RPC API of the logic layer.
type LogicLayerClient struct {
	Name string
	xmlrpc.Caller
}

// ListDevices retrieves the device descriptions from all devices.
func (c *LogicLayerClient) ListDevices(interfaceID string) ([]*DeviceDescription, error) {
	lclnLog.Debugf("Calling method listDevices(%s) on %s", interfaceID, c.Name)
	// execute call
	v, err := c.Call("listDevices", []*xmlrpc.Value{xmlrpc.NewString(interfaceID)})
	if err != nil {
		return nil, err
	}

	// build result
	e := xmlrpc.Q(v)
	// ReGaHss sends an empty value for an empty array. HMServer sends a correct
	// response.
	if e.IsEmpty() {
		return nil, nil
	}
	var r []*DeviceDescription
	for _, av := range e.Slice() {
		d := &DeviceDescription{}
		d.ReadFrom(av)
		r = append(r, d)
	}

	if e.Err() != nil {
		return nil, fmt.Errorf("Invalid XML response for listDevices: %v", e.Err())
	}
	return r, nil
}

// Event sends an event.
func (c *LogicLayerClient) Event(interfaceID, address, valueKey string, value interface{}) error {
	lclnLog.Debugf("Calling method event(%s, %s, %s, %v) on %s", interfaceID, address, valueKey, value, c.Name)
	// execute call
	v, err := xmlrpc.NewValue(value)
	if err != nil {
		return err
	}
	resp, err := c.Call("event", []*xmlrpc.Value{
		xmlrpc.NewString(interfaceID),
		xmlrpc.NewString(address),
		xmlrpc.NewString(valueKey),
		v,
	})
	if err != nil {
		return err
	}
	// assert empty response
	err = assertEmptyResponse(resp)
	if err != nil {
		return fmt.Errorf("Invalid response for method event: %v", err)
	}
	return nil
}

// NewDevices adds devices to the logic layer.
func (c *LogicLayerClient) NewDevices(interfaceID string, devDescriptions []*DeviceDescription) error {
	if lclnLog.DebugEnabled() {
		var addrs []string
		for _, dd := range devDescriptions {
			addrs = append(addrs, dd.Address)
		}
		lclnLog.Debugf("Calling method newDevices(%s, %s) on %s", interfaceID, strings.Join(addrs, " "), c.Name)
	}
	// parameters
	var data []*xmlrpc.Value
	for _, dd := range devDescriptions {
		data = append(data, dd.ToValue())
	}
	// execute call
	resp, err := c.Call("newDevices", []*xmlrpc.Value{
		xmlrpc.NewString(interfaceID),
		{Array: &xmlrpc.Array{Data: data}},
	})
	if err != nil {
		return err
	}
	// assert empty response
	err = assertEmptyResponse(resp)
	if err != nil {
		return fmt.Errorf("Invalid response for method newDevices: %v", err)
	}
	return nil
}

// DeleteDevices delete devicess from the logic layer.
func (c *LogicLayerClient) DeleteDevices(interfaceID string, addresses []string) error {
	lclnLog.Debugf("Calling method deleteDevices(%s, %s) on %s", interfaceID, strings.Join(addresses, " "), c.Name)
	// execute call
	resp, err := c.Call("deleteDevices", []*xmlrpc.Value{
		xmlrpc.NewString(interfaceID),
		xmlrpc.NewStrings(addresses),
	})
	if err != nil {
		return err
	}
	// assert empty response
	err = assertEmptyResponse(resp)
	if err != nil {
		return fmt.Errorf("Invalid response for method deleteDevices: %v", err)
	}
	return nil
}

func assertEmptyResponse(v *xmlrpc.Value) error {
	// empty array?
	if v.Array != nil {
		if len(v.Array.Data) != 0 {
			return errors.New("Array not empty")
		}
		return nil
	}
	// other types?
	if v.Boolean != "" || v.I4 != "" || v.Int != "" || v.Double != "" ||
		v.Base64 != "" || v.DateTime != "" || v.Struct != nil {
		return errors.New("Not a string or array")
	}
	// empty string?
	if v.ElemString != "" || v.FlatString != "" {
		return errors.New("String not empty")
	}
	return nil
}
