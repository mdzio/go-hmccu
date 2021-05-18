package itf

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

type logicLayer struct {
	msg chan string
}

func (l *logicLayer) Event(interfaceID, address, valueKey string, value interface{}) error {
	l.msg <- fmt.Sprintf("%s %s %s %v", interfaceID, address, valueKey, value)
	return nil
}

func (l *logicLayer) NewDevices(interfaceID string, devDescriptions []*DeviceDescription) error {
	var addrs []string
	for _, descr := range devDescriptions {
		addrs = append(addrs, descr.Address)
	}
	l.msg <- fmt.Sprintf("%s %v", interfaceID, addrs)
	return nil
}

func (l *logicLayer) DeleteDevices(interfaceID string, addresses []string) error {
	l.msg <- fmt.Sprintf("%s %v", interfaceID, addresses)
	return nil
}

func (l *logicLayer) UpdateDevice(interfaceID, address string, hint int) error {
	l.msg <- fmt.Sprintf("%s %s %d", interfaceID, address, hint)
	return nil
}

func (l *logicLayer) ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress string) error {
	l.msg <- fmt.Sprintf("%s %s %s", interfaceID, oldDeviceAddress, newDeviceAddress)
	return nil
}

func (l *logicLayer) ReaddedDevice(interfaceID string, deletedAddresses []string) error {
	l.msg <- fmt.Sprintf("%s %v", interfaceID, deletedAddresses)
	return nil
}

func TestLogicLayerServer(t *testing.T) {
	l := &logicLayer{msg: make(chan string, 1)}
	d := NewDispatcher()
	d.AddLogicLayer(l)
	h := &xmlrpc.Handler{Dispatcher: d}
	srv := httptest.NewServer(h)
	defer srv.Close()
	cln := &xmlrpc.Client{Addr: strings.TrimPrefix(srv.URL, "http://")}

	cases := []struct {
		want string
		call func() (*xmlrpc.Value, error)
	}{
		{
			"myid [ABC123 DEF456]",
			func() (*xmlrpc.Value, error) {
				return cln.Call("deleteDevices", []*xmlrpc.Value{
					{FlatString: "myid"},
					{
						Array: &xmlrpc.Array{
							Data: []*xmlrpc.Value{
								{FlatString: "ABC123"},
								{FlatString: "DEF456"},
							},
						},
					},
				})
			},
		},
		{
			"myid ABC123 3",
			func() (*xmlrpc.Value, error) {
				return cln.Call("updateDevice", []*xmlrpc.Value{
					{FlatString: "myid"},
					{FlatString: "ABC123"},
					{Int: "3"},
				})
			},
		},
		{
			"myid ABC123 DEF456",
			func() (*xmlrpc.Value, error) {
				return cln.Call("replaceDevice", []*xmlrpc.Value{
					{FlatString: "myid"},
					{FlatString: "ABC123"},
					{FlatString: "DEF456"},
				})
			},
		},
		{
			"myid [ABC123 DEF456]",
			func() (*xmlrpc.Value, error) {
				return cln.Call("readdedDevice", []*xmlrpc.Value{
					{FlatString: "myid"},
					{
						Array: &xmlrpc.Array{
							Data: []*xmlrpc.Value{
								{FlatString: "ABC123"},
								{FlatString: "DEF456"},
							},
						},
					},
				})
			},
		},
	}

	for no, c := range cases {
		res, err := c.call()
		if err != nil {
			t.Errorf("test case %d: unexpected client error: %v", no+1, err)
		}
		q := xmlrpc.Q(res)
		str := q.String()
		if q.Err() != nil || str != "" {
			t.Error("unexpected client result: ", res)
		}
		msg := <-l.msg
		if c.want != msg {
			t.Errorf("test case %d: unexpected callback result: %s", no+1, msg)
		}
	}
}

type deviceLayer struct{}

func (d *deviceLayer) Init(receiverAddress, interfaceID string) error {
	if receiverAddress == "http://abc" && interfaceID == "logicLayerID" {
		return nil
	}
	return &xmlrpc.MethodError{Code: 21, Message: "msg"}
}

func (d *deviceLayer) Deinit(receiverAddress string) error {
	if receiverAddress == "http://abc" {
		return nil
	}
	return errors.New("bad params")
}

func (d *deviceLayer) ListDevices() ([]*DeviceDescription, error) {
	return []*DeviceDescription{{Type: "MY-TYPE", Address: "ABC000000", RFAddress: 1}}, nil
}

func (d *deviceLayer) DeleteDevice(deviceAddress string, flags int) error {
	if deviceAddress != "ABC000000" || flags != 1 {
		return errors.New("bad params")
	}
	return nil
}

func (d *deviceLayer) GetDeviceDescription(deviceAddress string) (*DeviceDescription, error) {
	if deviceAddress != "ABC000000" {
		return nil, errors.New("bad params")
	}
	return &DeviceDescription{Type: "MY-TYPE", Address: "ABC000000", RFAddress: 1}, nil
}

func (d *deviceLayer) GetParamsetDescription(deviceAddress, paramsetType string) (ParamsetDescription, error) {
	if deviceAddress != "ABC000000:1" || paramsetType != "VALUES" {
		return nil, errors.New("bad params")
	}
	return ParamsetDescription{"PRESS_SHORT": {Type: "ACTION", Default: false, Min: false, Max: true}}, nil
}

func (d *deviceLayer) GetParamset(deviceAddress string, paramsetKey string) (map[string]interface{}, error) {
	if deviceAddress != "ABC000000:1" || paramsetKey != "MASTER" {
		return nil, errors.New("bad params")
	}
	return map[string]interface{}{"ARR_TIMEOUT": 123}, nil
}

func (d *deviceLayer) PutParamset(deviceAddress string, paramsetType string, paramset map[string]interface{}) error {
	if deviceAddress != "ABC000000:1" || paramsetType != "VALUES" ||
		!reflect.DeepEqual(paramset, map[string]interface{}{"LEVEL": 123}) {
		return errors.New("bad params")
	}
	return nil
}

func (d *deviceLayer) GetValue(deviceAddress string, valueName string) (interface{}, error) {
	if deviceAddress != "ABC000000:1" || valueName != "LEVEL" {
		return nil, errors.New("bad params")
	}
	return 123, nil
}

func (d *deviceLayer) SetValue(deviceAddress string, valueName string, value interface{}) error {
	if deviceAddress != "ABC000000:1" || valueName != "LEVEL" || value != 123 {
		return errors.New("bad params")
	}
	return nil
}

func (d *deviceLayer) Ping(callerID string) (bool, error) {
	if callerID != "abc" {
		return false, errors.New("bad params")
	}
	return true, nil
}

func TestDeviceLayerServer(t *testing.T) {
	dl := &deviceLayer{}
	di := NewDispatcher()
	di.AddDeviceLayer(dl)
	h := &xmlrpc.Handler{Dispatcher: di}
	srv := httptest.NewServer(h)
	defer srv.Close()
	cln := DeviceLayerClient{
		Name:   srv.URL,
		Caller: &xmlrpc.Client{Addr: strings.TrimPrefix(srv.URL, "http://")},
	}

	err := cln.Init("http://abc", "logicLayerID")
	if err != nil {
		t.Error(err)
	}
	err = cln.Init("http://force-error", "logicLayerID")
	if err == nil {
		t.Error("expected error")
	} else {
		if !reflect.DeepEqual(err, &xmlrpc.MethodError{Code: 21, Message: "msg"}) {
			t.Error(err)
		}
	}
	err = cln.Deinit("http://abc")
	if err != nil {
		t.Error(err)
	}

	dds, err := cln.ListDevices()
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(dds, []*DeviceDescription{{Type: "MY-TYPE", Address: "ABC000000", RFAddress: 1}}) {
		t.Error(dds)
	}

	err = cln.DeleteDevice("ABC000000", 1)
	if err != nil {
		t.Error(err)
	}

	dd, err := cln.GetDeviceDescription("ABC000000")
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(dd, &DeviceDescription{Type: "MY-TYPE", Address: "ABC000000", RFAddress: 1}) {
		t.Error(dd)
	}

	psd, err := cln.GetParamsetDescription("ABC000000:1", "VALUES")
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(psd, ParamsetDescription{"PRESS_SHORT": {Type: "ACTION", Default: false, Min: false, Max: true}}) {
		t.Error(psd)
	}

	ps, err := cln.GetParamset("ABC000000:1", "MASTER")
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(ps, map[string]interface{}{"ARR_TIMEOUT": 123}) {
		t.Error(ps)
	}

	err = cln.PutParamset("ABC000000:1", "VALUES", map[string]interface{}{"LEVEL": 123})
	if err != nil {
		t.Error(err)
	}

	v, err := cln.GetValue("ABC000000:1", "LEVEL")
	if err != nil {
		t.Error(err)
	} else if v != 123 {
		t.Error(v)
	}

	err = cln.SetValue("ABC000000:1", "LEVEL", 123)
	if err != nil {
		t.Error(err)
	}

	ret, err := cln.Ping("abc")
	if err != nil {
		t.Error(err)
	} else if ret != true {
		t.Error(ret)
	}
}
