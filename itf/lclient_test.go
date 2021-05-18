package itf

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

func TestLogicLayerClient(t *testing.T) {
	l := &logicLayer{msg: make(chan string, 1)}
	d := NewDispatcher()
	d.AddLogicLayer(l)
	h := &xmlrpc.Handler{Dispatcher: d}
	srv := httptest.NewServer(h)
	defer srv.Close()
	cln := LogicLayerClient{
		Name:   "LogicLayerClient",
		Caller: &xmlrpc.Client{Addr: strings.TrimPrefix(srv.URL, "http://")},
	}

	err := cln.NewDevices("itfID", []*DeviceDescription{
		{Address: "ABC00000"},
		{Address: "ABC00001"},
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := <-l.msg
	if msg != "itfID [ABC00000 ABC00001]" {
		t.Fatal("NewDevices invalid")
	}

	err = cln.DeleteDevices("itfID", []string{"ABC00000", "ABC00001"})
	if err != nil {
		t.Fatal(err)
	}
	msg = <-l.msg
	if msg != "itfID [ABC00000 ABC00001]" {
		t.Fatal("DeleteDevices invalid")
	}
}
