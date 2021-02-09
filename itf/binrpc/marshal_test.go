package binrpc

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

func TestEncodeRequest(t *testing.T) {
	cases := []struct {
		method string
		params []*xmlrpc.Value
		want   string
	}{
		{
			"system.listMethods",
			[]*xmlrpc.Value{},
			"42 69 6e 00 00 00 00 1a 00 00 00 12 73 79 73 74 65 6d 2e 6c 69 73 74 4d 65 74 68 6f 64 73 00 00 00 00",
		},
		{
			"init",
			[]*xmlrpc.Value{
				{
					ElemString: "xmlrpc_bin://172.16.23.180:2004",
				},
				{
					ElemString: "test",
				},
			},
			"42 69 6e 00 00 00 00 3f 00 00 00 04 69 6e 69 74 00 00 00 02 00 00 00 03 00 00 00 1f 78 6d 6c 72 70 63 5f 62 69 6e 3a 2f 2f 31 37 32 2e 31 36 2e 32 33 2e 31 38 30 3a 32 30 30 34 00 00 00 03 00 00 00 04 74 65 73 74",
		},
	}

	for _, tt := range cases {
		t.Run(tt.method, func(t *testing.T) {
			buf := bytes.Buffer{}
			e := NewEncoder(&buf)
			err := e.EncodeRequest(tt.method, tt.params)
			if err != nil {
				t.Error(err)
			}
			out, err := ioutil.ReadAll(&buf)
			if err != nil {
				t.Error(err)
			}
			want := strings.ReplaceAll(tt.want, " ", "")
			got := hex.EncodeToString(out)
			if got != want {
				t.Errorf("Expected: %s, got: %s", want, got)
			}
		})
	}
}

func TestEncodeParam(t *testing.T) {
	tests := []struct {
		name    string
		in      xmlrpc.Value
		out     string
		wantErr bool
	}{
		{
			"String BidCoS-RF",
			xmlrpc.Value{
				ElemString: "BidCoS-RF",
			},
			"00 00 00 03 00 00 00 09 42 69 64 43 6f 53 2d 52 46",
			false,
		},
		{
			"Integer 41",
			xmlrpc.Value{
				Int: "41",
			},
			"00 00 00 01 00 00 00 29",
			false,
		},
		{
			"Integer xx",
			xmlrpc.Value{
				Int: "xx",
			},
			"",
			true,
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
			"Bool xx",
			xmlrpc.Value{
				Boolean: "xx",
			},
			"",
			true,
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
		{
			"Double xx",
			xmlrpc.Value{
				Double: "xx",
			},
			"",
			true,
		},
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
		{
			"Array 41 41",
			xmlrpc.Value{
				Array: &xmlrpc.Array{
					Data: []*xmlrpc.Value{
						{
							Int: "41",
						},
						{
							Int: "41",
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
			buf := bytes.Buffer{}
			e := NewEncoder(&buf)
			err := e.encodeParams([]*xmlrpc.Value{&tt.in})
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error in case %s", tt.name)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			out, err := ioutil.ReadAll(e.paramBuf)
			if err != nil {
				t.Error(err)
			}
			want := strings.ReplaceAll(tt.out, " ", "")
			got := hex.EncodeToString(out)
			if got != want {
				t.Errorf("Expected: %s, got: %s", want, got)
			}
		})
	}
}
