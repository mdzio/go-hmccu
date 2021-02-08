package xmlrpc

import (
	"testing"

	"github.com/mdzio/go-lib/testutil"
)

// Test configuration (environment variables)
const (
	// LOG_LEVEL: OFF, ERROR, WARNING, INFO, DEBUG, TRACE

	// hostname or IP address of the test CCU, e.g. 192.168.0.10
	ccuAddress = "CCU_ADDRESS"
)

func itfAddress(t *testing.T) string {
	// use BidCos-RF XML-RPC interface
	return "http://" + testutil.Config(t, ccuAddress) + ":2001"
}

func TestClient_Call(t *testing.T) {
	ccuAddress := itfAddress(t)
	c := Client{Addr: ccuAddress}

	d, err := c.Call("unknownMethod", []*Value{})
	if d != nil || err == nil {
		t.Error("error expected")
	}
	if err.Error() != "RPC fault (code: -1, message: unknownMethod: unknown method name)" {
		t.Errorf("unexpected error: %v", err)
	}

	d, err = c.Call("getDeviceDescription", []*Value{{FlatString: "BidCoS-RF:0"}})
	if err != nil {
		t.Fatal(err)
	}
	e := Q(d)
	str := e.Key("ADDRESS").String()
	if str != "BidCoS-RF:0" {
		t.Error("invalid ADDRESS")
	}
	str = e.Key("PARENT_TYPE").String()
	if str != "HM-RCV-50" {
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
