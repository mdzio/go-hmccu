/*
Environment variables for integration tests:
	CCU_ADDRESS:
		hostname or IP address of the test CCU2
	LOG_LEVEL:
		off, error, warning, info, debug, trace
*/
package binrpc

import (
	"os"
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
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
	ccuAddr := os.Getenv("CCU_ADDRESS")
	if len(ccuAddr) == 0 {
		t.Skip("environment variable CCU_ADDRESS not set")
	}
	// use BidCos-RF XML-RPC interface
	return ccuAddr + ":8701"
}

func TestClient_Call(t *testing.T) {
	ccuAddress := getXMLRPCAddr(t)
	c := Client{Addr: ccuAddress}

	//d, err := c.Call("unknownMethod", []*xmlrpc.Value{})
	//if d != nil || err == nil {
	//	t.Error("error expected")
	//}
	//if err.Error() != "XML-RPC fault (code: -1, message: unknownMethod: unknown method name)" {
	//	t.Errorf("unexpected error: %v", err)
	//}

	d, err := c.Call("getDeviceDescription", []*xmlrpc.Value{{FlatString: "CUX1200002:1"}})
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
