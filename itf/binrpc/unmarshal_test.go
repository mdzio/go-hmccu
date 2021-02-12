package binrpc

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

func TestDecodeRequest(t *testing.T) {
	in := strings.ReplaceAll("42 69 6e 00 00 00 00 3f 00 00 00 04 69 6e 69 74 00 00 00 02 00 00 00 03 00 00 00 1f 78 6d 6c 72 "+
		"70 63 5f 62 69 6e 3a 2f 2f 31 37 32 2e 31 36 2e 32 33 2e 31 38 30 3a 32 30 30 34 00 00 00 03 00 00 00 04 74 65 73 74", " ", "")
	b, err := hex.DecodeString(in)
	if err != nil {
		t.Fatal()
	}
	d := NewDecoder(bytes.NewReader(b))
	method, params, err := d.DecodeRequest()
	if err != nil {
		t.Error(err)
	}
	if method != "init" {
		t.Errorf("Unexpected method name: %s", method)
	}
	want := xmlrpc.Values{
		{FlatString: "xmlrpc_bin://172.16.23.180:2004"},
		{FlatString: "test"},
	}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("Unexpected params: %s", params)
	}
}

func TestDecodeValue(t *testing.T) {
	tests := []struct {
		name string
		val  *xmlrpc.Value
	}{
		{
			"String üöäÜÖÄß",
			&xmlrpc.Value{FlatString: "üöäÜÖÄß"},
		},
		{
			"Integer 41",
			&xmlrpc.Value{I4: "41"},
		},
		{
			"Bool 0",
			&xmlrpc.Value{Boolean: "0"},
		},
		{
			"Bool 1",
			&xmlrpc.Value{Boolean: "1"},
		},
		{
			"Double 1234",
			&xmlrpc.Value{Double: "1234.000000"},
		},
		{
			"Double -9999.015625",
			&xmlrpc.Value{Double: "-9999.015625"},
		},
		{
			"Double 0.000001",
			&xmlrpc.Value{Double: "0.000001"},
		},
		{
			"Array",
			&xmlrpc.Value{Array: &xmlrpc.Array{Data: []*xmlrpc.Value{
				{FlatString: "abc"},
				{I4: "-999"},
			}}},
		},
		{
			"Struct",
			&xmlrpc.Value{Struct: &xmlrpc.Struct{Members: []*xmlrpc.Member{
				{Name: "a", Value: &xmlrpc.Value{Boolean: "0"}},
				{Name: "b", Value: &xmlrpc.Value{Double: "125.125000"}},
				{Name: "c", Value: &xmlrpc.Value{I4: "125"}},
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// encode
			e := valueEncoder{}
			err := e.encodeValue(tt.val)
			if err != nil {
				t.Fatal(err)
			}
			// decode
			r := bytes.NewReader(e.Bytes())
			d := NewDecoder(r)
			val, err := d.decodeValue()
			if err != nil {
				t.Error(err)
			}
			// compare
			if !reflect.DeepEqual(tt.val, val) {
				t.Errorf("Expected: %v Got: %v", tt.val, val)
			}
		})
	}
}
