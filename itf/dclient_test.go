package itf

import (
	"reflect"
	"testing"

	"github.com/mdzio/go-hmccu/v2/itf/xmlrpc"
	"github.com/mdzio/go-hmccu/v2/testcfg"
)

// Test configuration (environment variables)
const (
	// LOG_LEVEL: ERROR, WARN, INFO, DEBUG

	// hostname or IP address of the test CCU, e.g. 192.168.0.10
	ccuAddress = "CCU_ADDRESS"

	// device address of a HM-LC-Sw1 (rf switch actor) ATTENTION: state will be
	// changed!
	hmlcsw1Device = "HMLCSW1_DEVICE"

	// device address of a HM-ES-PMSw1 (rf switch actor with meter) ATTENTION:
	// TRANSMIT_TRY_MAX from parameter set MASTER will be changed!
	hmespmsw1Device = "HMESPMSW1_DEVICE"
)

func newXMLTestClient(t *testing.T) *DeviceLayerClient {
	addr := testcfg.Config(t, ccuAddress) + ":2001"
	return &DeviceLayerClient{
		Name:   addr,
		Caller: &xmlrpc.Client{Addr: addr},
	}
}

func TestClient_GetDeviceDescription(t *testing.T) {
	c := newXMLTestClient(t)

	d, err := c.GetDeviceDescription("BidCoS-RF:0")
	if err != nil {
		t.Fatal(err)
	}
	want := &DeviceDescription{
		Type:       "MAINTENANCE",
		Address:    "BidCoS-RF:0",
		Parent:     "BidCoS-RF",
		ParentType: "HM-RCV-50",
		Paramsets:  []string{"MASTER", "VALUES"},
		Version:    6,
		Flags:      3,
	}
	if !reflect.DeepEqual(*d, *want) {
		t.Error("Result does not match")
	}
}

func TestClient_ListDevices(t *testing.T) {
	c := newXMLTestClient(t)

	_, err := c.ListDevices()
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_GetParamsetDescription(t *testing.T) {
	c := newXMLTestClient(t)

	ps, err := c.GetParamsetDescription("BidCoS-RF:1", "VALUES")
	if err != nil {
		t.Fatal(err)
	}
	want := &ParameterDescription{
		Control:    "BUTTON.SHORT",
		Default:    false,
		Flags:      1,
		ID:         "PRESS_SHORT",
		Max:        true,
		Min:        false,
		Operations: 6,
		TabOrder:   1,
		Type:       "ACTION",
		Unit:       "",
	}
	if !reflect.DeepEqual(*ps["PRESS_SHORT"], *want) {
		t.Error("Result does not match")
	}
}

func TestClient_GetParamset(t *testing.T) {
	c := newXMLTestClient(t)

	ps, err := c.GetParamset(testcfg.Config(t, hmlcsw1Device)+":1", "VALUES")
	if err != nil {
		t.Fatal(err)
	}

	members := []string{"INHIBIT", "STATE", "WORKING"}
	for _, m := range members {
		v, ok := ps[m]
		if !ok {
			t.Errorf("missing member: %s", m)
		}
		_, ok = v.(bool)
		if !ok {
			t.Errorf("not a bool: %s", m)
		}
	}
}

func TestClient_GetSetParamsetMaster(t *testing.T) {
	c := newXMLTestClient(t)

	ps, err := c.GetParamset(testcfg.Config(t, hmespmsw1Device)+":1", "MASTER")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := ps["TRANSMIT_TRY_MAX"]
	if !ok {
		t.Fatal("TRANSMIT_TRY_MAX not found in paramset")
	}
	tryMax, ok := v.(int)
	if !ok {
		t.Fatal("TRANSMIT_TRY_MAX is not an int")
	}

	err = c.PutParamset(
		testcfg.Config(t, hmespmsw1Device)+":1",
		"MASTER",
		map[string]interface{}{"TRANSMIT_TRY_MAX": tryMax + 1},
	)
	if err != nil {
		t.Fatal(err)
	}

	// restore previous value
	err = c.PutParamset(
		testcfg.Config(t, hmespmsw1Device)+":1",
		"MASTER",
		map[string]interface{}{"TRANSMIT_TRY_MAX": tryMax},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_GetSetValue(t *testing.T) {
	c := newXMLTestClient(t)

	val, err := c.GetValue(testcfg.Config(t, hmlcsw1Device)+":1", "STATE")
	if err != nil {
		t.Fatal(err)
	}
	b, ok := val.(bool)
	if !ok {
		t.Fatal("bool expected")
	}

	err = c.SetValue(testcfg.Config(t, hmlcsw1Device)+":1", "STATE", b)
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_Deinit(t *testing.T) {
	c := newXMLTestClient(t)

	err := c.Deinit("http://unknownAddress")
	if err != nil {
		t.Fatal(err)
	}
}
