package binrpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

func TestDecodeRequest(t *testing.T) {
	in := strings.ReplaceAll("42 69 6e 00 00 00 00 3f 00 00 00 04 69 6e 69 74 00 00 00 02 00 00 00 03 00 00 00 1f 78 6d 6c 72 70 63 5f 62 69 6e 3a 2f 2f 31 37 32 2e 31 36 2e 32 33 2e 31 38 30 3a 32 30 30 34 00 00 00 03 00 00 00 04 74 65 73 74", " ", "")
	b, err := hex.DecodeString(in)
	if err != nil {
		t.Errorf("Failed to decode string")
	}
	r := bytes.NewReader(b)
	d := NewDecoder(r)
	_, vals, err := d.DecodeRequest()

	for _, val := range vals {
		fmt.Printf("Values: %#v\n", *val)
	}
}

func TestDecodeParam(t *testing.T) {
	tests := []struct {
		name    string
		in      xmlrpc.Value
		out     string
		wantErr bool
	}{
		{
			"String BidCoS-RF",
			xmlrpc.Value{
				FlatString: "BidCoS-RF",
			},
			"00 00 00 03 00 00 00 09 42 69 64 43 6f 53 2d 52 46",
			false,
		},
		{
			"Integer 41",
			xmlrpc.Value{
				I4: "41",
			},
			"00 00 00 01 00 00 00 29",
			false,
		},

		{
			"Bool 0",
			xmlrpc.Value{
				Boolean: "0",
			},
			"00 00 00 02 00",
			false,
		},
		{
			"Bool 1",
			xmlrpc.Value{
				Boolean: "1",
			},
			"00 00 00 02 01",
			false,
		},
		{
			"Double 1234",
			xmlrpc.Value{
				Double: "1234",
			},
			"00 00 00 04 26 90 00 00 00 00 00 0b",
			false,
		},
		{
			"Double -9999.9999",
			xmlrpc.Value{
				Double: "-9999.9999",
			},
			"00 00 00 04 d8 f0 00 06 00 00 00 0e",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := valueEncoder{}
			err := e.encodeParams([]*xmlrpc.Value{&tt.in})
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			r := bytes.NewReader(e.Bytes())
			d := NewDecoder(r)
			vals, err := d.decodeParamValues(1)
			if len(vals) == 0 {
				t.Errorf("Failed to decode values: %w", err)
				return
			}
			if !reflect.DeepEqual(tt.in, *vals[0]) {
				t.Error("Unexpected value")
			}
		})
	}
}

func TestDecodeArrayParam(t *testing.T) {
	tests := []struct {
		name    string
		in      xmlrpc.Value
		out     string
		wantErr bool
	}{
		{
			"Array 41 41",
			xmlrpc.Value{
				Array: &xmlrpc.Array{
					Data: []*xmlrpc.Value{
						{
							I4: "41",
						},
						{
							I4: "41",
						},
					},
				},
			},
			"00 00 01 00 00 00 00 02 00 00 00 01 00 00 00 29 00 00 00 01 00 00 00 29",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := valueEncoder{}
			err := e.encodeParams([]*xmlrpc.Value{&tt.in})
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			r := bytes.NewReader(e.Bytes())
			d := NewDecoder(r)
			vals, err := d.decodeParamValues(1)
			if len(vals) == 0 {
				t.Errorf("Failed to decode values: %w", err)
				return
			}

			for i := 0; i < len(vals[0].Array.Data); i++ {
				if !reflect.DeepEqual(tt.in.Array.Data[i], vals[0].Array.Data[i]) {
					t.Error("Unexpected value")
				}
			}
		})
	}
}

func TestDecodeStructParam(t *testing.T) {
	tests := []struct {
		name    string
		in      xmlrpc.Value
		out     string
		wantErr bool
	}{
		{
			"Struct {'Temperature': 20.5}",
			xmlrpc.Value{
				Struct: &xmlrpc.Struct{Members: []*xmlrpc.Member{
					{
						Name: "Temperature",
						Value: &xmlrpc.Value{
							Double: "20.5",
						},
					},
				}},
			},
			"00 00 01 01 00 00 00 01 00 00 00 0b 54 65 6d 70 65 72 61 74 75 72 65 00 00 00 04 29 00 00 00 00 00 00 05",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := valueEncoder{}
			err := e.encodeParams([]*xmlrpc.Value{&tt.in})
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			r := bytes.NewReader(e.Bytes())
			d := NewDecoder(r)
			vals, err := d.decodeParamValues(1)
			if len(vals) == 0 {
				t.Errorf("Failed to decode values: %w", err)
				return
			}

			for i := 0; i < len(vals[0].Struct.Members); i++ {
				if !reflect.DeepEqual(tt.in.Struct.Members[i], vals[0].Struct.Members[i]) {
					t.Error("Unexpected result")
				}
			}
		})
	}
}
