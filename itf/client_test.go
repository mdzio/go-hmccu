/*
Environment variables for integration tests:
	CCU_ADDRESS:
		hostname or IP address of the test CCU2
	LOCAL_ADDRESS:
		hostname or IP address of the test machine (for callbacks)
	HMLCSW1_DEVICE:
		device address of a HM-LC-Sw1 (rf switch actor)
		attention: state will be changed!
	HMESPMSW1_DEVICE:
		device address of a HM-ES-PMSw1 (rf switch actor with meter)
		attention: TRANSMIT_TRY_MAX from parameter set MASTER will be changed!
	LOG_LEVEL:
		off, error, warning, info, debug, trace
*/
package itf

import (
	"os"
	"reflect"
	"testing"

	"github.com/mdzio/go-lib/any"
	"github.com/mdzio/go-logging"
)

func init() {
	var l logging.LogLevel
	err := l.Set(os.Getenv("LOG_LEVEL"))
	if err == nil {
		logging.SetLevel(l)
	}
}

func getXMLRPCAddr(t *testing.T) string {
	addr := os.Getenv("CCU_ADDRESS")
	if len(addr) == 0 {
		t.Skip("environment variable CCU_ADDRESS not set")
	}
	return "http://" + addr + ":2001"
}

func getLocalAddr(t *testing.T) string {
	addr := os.Getenv("LOCAL_ADDRESS")
	if len(addr) == 0 {
		t.Skip("environment variable LOCAL_ADDRESS not set")
	}
	return "http://" + addr
}

func getHMLCSW1Device(t *testing.T) string {
	hmlcsw1Device := os.Getenv("HMLCSW1_DEVICE")
	if len(hmlcsw1Device) == 0 {
		t.Skip("environment variable HMLCSW1_DEVICE not set")
	}
	return hmlcsw1Device
}

func getHMESPMSW1Device(t *testing.T) string {
	hmlcsw1Device := os.Getenv("HMESPMSW1_DEVICE")
	if len(hmlcsw1Device) == 0 {
		t.Skip("environment variable HMESPMSW1_DEVICE not set")
	}
	return hmlcsw1Device
}

func TestClient_GetDeviceDescription(t *testing.T) {
	c := NewClient(getXMLRPCAddr(t))

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
	c := NewClient(getXMLRPCAddr(t))

	_, err := c.ListDevices()
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_GetParamsetDescription(t *testing.T) {
	c := NewClient(getXMLRPCAddr(t))

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
	c := NewClient(getXMLRPCAddr(t))

	ps, err := c.GetParamset(getHMLCSW1Device(t)+":1", "VALUES")
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
	c := NewClient(getXMLRPCAddr(t))

	ps, err := c.GetParamset(getHMESPMSW1Device(t)+":1", "MASTER")
	if err != nil {
		t.Fatal(err)
	}
	psm := any.Q(ps).Map()
	tryMax := psm.Key("TRANSMIT_TRY_MAX").Int()
	if psm.Err() != nil {
		t.Fatal(err)
	}

	err = c.PutParamset(
		getHMESPMSW1Device(t)+":1",
		"MASTER",
		map[string]interface{}{"TRANSMIT_TRY_MAX": tryMax + 1},
	)
	if err != nil {
		t.Fatal(err)
	}

	// restore previous value
	err = c.PutParamset(
		getHMESPMSW1Device(t)+":1",
		"MASTER",
		map[string]interface{}{"TRANSMIT_TRY_MAX": tryMax},
	)
}

func TestClient_GetSetValue(t *testing.T) {
	c := NewClient(getXMLRPCAddr(t))

	val, err := c.GetValue(getHMLCSW1Device(t)+":1", "STATE")
	if err != nil {
		t.Fatal(err)
	}
	b, ok := val.(bool)
	if !ok {
		t.Fatal("bool expected")
	}

	err = c.SetValue(getHMLCSW1Device(t)+":1", "STATE", b)
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_Deinit(t *testing.T) {
	c := NewClient(getXMLRPCAddr(t))

	err := c.Deinit("http://unknownAddress")
	if err != nil {
		t.Fatal(err)
	}
}
