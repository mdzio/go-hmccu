package itf

import (
	"fmt"
	"github.com/mdzio/go-hmccu/model"
	"net/http/httptest"
	"testing"
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
	h := NewHandler(r)
	srv := httptest.NewServer(h)
	defer srv.Close()

	cln := NewClient(srv.URL)

	cases := []struct {
		want string
		call func() (*model.Value, error)
	}{
		{
			"interfaceID address valueKey 123.456",
			func() (*model.Value, error) {
				return nil, cln.Event("interfaceID", "address", "valueKey", 123.456)
			},
		},
		{
			"myid [ABC123 DEF456]",
			func() (*model.Value, error) {
				return cln.Call("deleteDevices", []*model.Value{
					&model.Value{FlatString: "myid"},
					&model.Value{
						Array: &model.Array{
							Data: []*model.Value{
								&model.Value{FlatString: "ABC123"},
								&model.Value{FlatString: "DEF456"},
							},
						},
					},
				})
			},
		},
		{
			"myid ABC123 3",
			func() (*model.Value, error) {
				return cln.Call("updateDevice", []*model.Value{
					&model.Value{FlatString: "myid"},
					&model.Value{FlatString: "ABC123"},
					&model.Value{Int: "3"},
				})
			},
		},
		{
			"myid ABC123 DEF456",
			func() (*model.Value, error) {
				return cln.Call("replaceDevice", []*model.Value{
					&model.Value{FlatString: "myid"},
					&model.Value{FlatString: "ABC123"},
					&model.Value{FlatString: "DEF456"},
				})
			},
		},
		{
			"myid [ABC123 DEF456]",
			func() (*model.Value, error) {
				return cln.Call("readdedDevice", []*model.Value{
					&model.Value{FlatString: "myid"},
					&model.Value{
						Array: &model.Array{
							Data: []*model.Value{
								&model.Value{FlatString: "ABC123"},
								&model.Value{FlatString: "DEF456"},
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
		q := model.Q(res)
		str := q.String()
		if q.Err() != nil || str != "" {
			t.Error("unexpected client result: ", res)
		}
		if c.want != r.msg {
			t.Errorf("test case %d: unexpected callback result: %s", no+1, r.msg)
		}
	}
}
