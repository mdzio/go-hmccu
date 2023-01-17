package itf

import (
	"encoding/xml"
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

func TestAssertEmptyResponse(t *testing.T) {
	cases := []struct {
		xml string
		err string
	}{
		{
			// response from CCU interface processes
			`<?xml version="1.0"?><methodResponse><params><param><value></value></param></params></methodResponse>`,
			"",
		},
		{
			`<?xml version="1.0"?><methodResponse><params><param><value>abc</value></param></params></methodResponse>`,
			"String not empty",
		},
		{
			// response from HAP-HomeMatic Add-On
			`<?xml version="1.0"?><methodResponse><params><param><value><array><data></data></array></value></param></params></methodResponse>`,
			"",
		},
		{
			`<?xml version="1.0"?><methodResponse><params><param><value><array><data><value></value></data></array></value></param></params></methodResponse>`,
			"Array not empty",
		},
		{
			`<?xml version="1.0"?><methodResponse><params><param><value><i4>123</i4></value></param></params></methodResponse>`,
			"Not a string or array",
		},
	}
	for _, c := range cases {
		dec := xml.NewDecoder(strings.NewReader(c.xml))
		resp := &xmlrpc.MethodResponse{}
		err := dec.Decode(resp)
		if err != nil {
			t.Fatal(err)
		} else {
			val := resp.Params.Param[0].Value
			err = assertEmptyResponse(val)
			if (err == nil && c.err == "") || (err != nil && err.Error() == c.err) {
				continue
			}
			t.Error(c.err, ":", err)
		}
	}
}
