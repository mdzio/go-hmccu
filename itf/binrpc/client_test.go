package binrpc

import (
	"strings"
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

	// test unknown method
	d, err := c.Call("unknownMethod", []*xmlrpc.Value{})
	if d != nil || err == nil {
		t.Error("error expected")
		// the dot in unknown.method is CUxD specific
	} else if err.Error() != "RPC fault (code: -1, message: unknownMethod: unknown.method name)" {
		t.Errorf("unexpected error: %v", err)
	}

	// test unknown instance
	d, err = c.Call("getDeviceDescription", []*xmlrpc.Value{{FlatString: "ZZZ9999999:1"}})
	if err == nil {
		t.Error("error expected")
	} else if err.Error() != "RPC fault (code: -2, message: Unknown instance)" {
		t.Errorf("unexpected error: %v", err)
	}

	// test response size limit
	c.ResponseSizeLimit = 1
	d, err = c.Call("getDeviceDescription", []*xmlrpc.Value{{FlatString: "CUX4000100:1"}})
	if d != nil || err == nil {
		t.Error("error expected")
	} else if !strings.HasSuffix(err.Error(), "unexpected EOF") {
		t.Errorf("unexpected error: %v", err)
	}
	c.ResponseSizeLimit = 0

	// test successful call
	d, err = c.Call("getDeviceDescription", []*xmlrpc.Value{{FlatString: "CUX4000100:1"}})
	if err != nil {
		t.Fatal(err)
	}
	e := xmlrpc.Q(d)
	str := e.Key("ADDRESS").String()
	if str != "CUX4000100:1" {
		t.Errorf("invalid ADDRESS: %s", str)
	}
	str = e.Key("PARENT_TYPE").String()
	if str != "HM-RC-19" {
		t.Errorf("invalid PARENT_TYPE: %s", str)
	}
	arr := e.Key("PARAMSETS").Strings()
	if len(arr) != 2 || arr[0] != "MASTER" || arr[1] != "VALUES" {
		t.Errorf("invalid PARAMSETS: %v", arr)
	}
	if e.Err() != nil {
		t.Error(e.Err())
	}

	// test another call
	d, err = c.Call("getDeviceDescription", []*xmlrpc.Value{{FlatString: "CUX4000101:1"}})
	if err != nil {
		t.Fatal(err)
	}
	e = xmlrpc.Q(d)
	str = e.Key("ADDRESS").String()
	if str != "CUX4000101:1" {
		t.Errorf("invalid ADDRESS: %s", str)
	}
	str = e.Key("PARENT_TYPE").String()
	if str != "HM-LC-Sw1-Pl" {
		t.Errorf("invalid PARENT_TYPE: %s", str)
	}
	arr = e.Key("PARAMSETS").Strings()
	if len(arr) != 2 || arr[0] != "MASTER" || arr[1] != "VALUES" {
		t.Errorf("invalid PARAMSETS: %v", arr)
	}
	if e.Err() != nil {
		t.Error(e.Err())
	}
}
