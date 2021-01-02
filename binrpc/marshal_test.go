package binrpc

import (
	"bytes"
	"encoding/hex"
	"github.com/mdzio/go-hmccu/model"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"strings"
	"testing"
)

func TestEncodeRequest(t *testing.T) {
	cases := []struct {
		method string
		params []*model.Value
		want   string
	}{
		{
			"system.listMethods",
			[]*model.Value{},
			"42 69 6e 00 00 00 00 1a 00 00 00 12 73 79 73 74 65 6d 2e 6c 69 73 74 4d 65 74 68 6f 64 73 00 00 00 00",
		},
		{
			"init",
			[]*model.Value{
				{
					String: "xmlrpc_bin://172.16.23.180:2004",
				},
				{
					String: "test",
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

			assert.Equal(t, hex.EncodeToString(out), strings.ReplaceAll(tt.want, " ", ""))
		})

	}
}

func TestEncodeParam(t *testing.T) {
	tests := []struct {
		name    string
		in      model.Value
		out     string
		wantErr bool
	}{
		{
			"String BidCoS-RF",
			model.Value{
				String: "BidCoS-RF",
			},
			"00 00 00 03 00 00 00 09 42 69 64 43 6f 53 2d 52 46",
			false,
		},
		{
			"Integer 41",
			model.Value{
				Int: "41",
			},
			"00 00 00 01 00 00 00 29",
			false,
		},
		{
			"Integer xx",
			model.Value{
				Int: "xx",
			},
			"",
			true,
		},

		{
			"Bool 0",
			model.Value{
				Boolean: "0",
			},
			"00 00 00 02 00",
			false,
		},
		{
			"Bool 1",
			model.Value{
				Boolean: "1",
			},
			"00 00 00 02 01",
			false,
		},
		{
			"Bool xx",
			model.Value{
				Boolean: "xx",
			},
			"",
			true,
		},
		{
			"Double 1234",
			model.Value{
				Double: "1234",
			},
			"00 00 00 04 26 90 00 00 00 00 00 0b",
			false,
		},
		{
			"Double -9999.9999",
			model.Value{
				Double: "-9999.9999",
			},
			"00 00 00 04 d8 f0 00 06 00 00 00 0e",
			false,
		},
		{
			"Double xx",
			model.Value{
				Double: "xx",
			},
			"",
			true,
		},
		{
			"Struct {'Temperature': 20.5}",
			model.Value{
				Struct: &model.Struct{Members: []*model.Member{
					&model.Member{
						Name: "Temperature",
						Value: &model.Value{
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
			model.Value{
				Array: &model.Array{
					Data: []*model.Value{
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
			err := e.encodeParams([]*model.Value{&tt.in})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			out, err := ioutil.ReadAll(e.paramBuf)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, strings.ReplaceAll(tt.out, " ", ""), hex.EncodeToString(out))
		})
	}
}
