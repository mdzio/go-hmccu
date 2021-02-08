package binrpc

import (
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"github.com/mdzio/go-lib/testutil"
)

// Test configuration (environment variables)
const (
	// LOG_LEVEL: OFF, ERROR, WARNING, INFO, DEBUG, TRACE

	// hostname or IP address of the test CCU, e.g. 192.168.0.10
	ccuAddress = "CCU_ADDRESS"
	// Open port 8701 in the CCU firewall.

	// Create two devices of type 40 in CUxD:
	// CUX4000100 (X) HM-RC-19 CUX4000100     * KEY
	// CUX4000101 (X) HM-LC-Sw1-Pl CUX4000101 * SWITCH
)

func newTestClient(t *testing.T) *Client {
	// use CUxD BIN-RPC interface
	return &Client{Addr: testutil.Config(t, ccuAddress) + ":8701"}
}

func TestClient_Call(t *testing.T) {
	c := newTestClient(t)

	d, err := c.Call("unknownMethod", []*xmlrpc.Value{})
	if d != nil || err == nil {
		t.Error("error expected")
		// the dot in unknown.method is CUxD specific
	} else if err.Error() != "RPC fault (code: -1, message: unknownMethod: unknown.method name)" {
		t.Errorf("unexpected error: %v", err)
	}

	d, err = c.Call("getDeviceDescription", []*xmlrpc.Value{{FlatString: "ZZZ9999999:1"}})
	if err == nil {
		t.Error("error expected")
	} else if err.Error() != "RPC fault (code: -2, message: Unknown instance)" {
		t.Errorf("unexpected error: %v", err)
	}

	d, err = c.Call("getDeviceDescription", []*xmlrpc.Value{{FlatString: "CUX1200002:1"}})
	if err != nil {
		t.Fatal(err)
	}
	e := xmlrpc.Q(d)
	str := e.Key("ADDRESS").String()
	if str != "CUX1200002:1" {
		t.Error("invalid ADDRESS")
	}
	str = e.Key("PARENT_TYPE").String()
	if str != "HM-WS550STH-I" {
		t.Error("invalid PARENT_TYPE")
	}
	arr := e.Key("PARAMSETS").Slice()
	if len(arr) != 2 || arr[0].String() != "MASTER" || arr[1].String() != "VALUES" {
		t.Error("invalid PARAMSETS")
	}
	if e.Err() != nil {
		t.Error(e.Err())
	}
}
