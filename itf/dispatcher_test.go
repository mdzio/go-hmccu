package itf

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

type receiver struct {
	msg string
}

func (r *receiver) Event(interfaceID, address, valueKey string, value interface{}) error {
	r.msg = fmt.Sprintf("%s %s %s %v", interfaceID, address, valueKey, value)
	return nil
}

func (r *receiver) NewDevices(interfaceID string, devDescriptions []*DeviceDescription) error {
	var addrs []string
	for _, descr := range devDescriptions {
		addrs = append(addrs, descr.Address)
	}
	r.msg = fmt.Sprintf("%s %v", interfaceID, addrs)
	return nil
}

func (r *receiver) DeleteDevices(interfaceID string, addresses []string) error {
	r.msg = fmt.Sprintf("%s %v", interfaceID, addresses)
	return nil
}

func (r *receiver) UpdateDevice(interfaceID, address string, hint int) error {
	r.msg = fmt.Sprintf("%s %s %d", interfaceID, address, hint)
	return nil
}

func (r *receiver) ReplaceDevice(interfaceID, oldDeviceAddress, newDeviceAddress string) error {
	r.msg = fmt.Sprintf("%s %s %s", interfaceID, oldDeviceAddress, newDeviceAddress)
	return nil
}

func (r *receiver) ReaddedDevice(interfaceID string, deletedAddresses []string) error {
	r.msg = fmt.Sprintf("%s %v", interfaceID, deletedAddresses)
	return nil
}

func TestServer(t *testing.T) {
	r := &receiver{}
	h := &xmlrpc.Handler{Dispatcher: NewDispatcher(r)}
	srv := httptest.NewServer(h)
	defer srv.Close()

	cln := Client{
		Name:   srv.URL,
		Caller: &xmlrpc.Client{Addr: srv.URL},
	}

	cases := []struct {
		want string
		call func() (*xmlrpc.Value, error)
	}{
		{
			"interfaceID address valueKey 123.456",
			func() (*xmlrpc.Value, error) {
				return nil, cln.Event("interfaceID", "address", "valueKey", 123.456)
			},
		},
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
		if c.want != r.msg {
			t.Errorf("test case %d: unexpected callback result: %s", no+1, r.msg)
		}
	}
}
