package itf

import (
	"reflect"
	"testing"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

func TestDeviceDescription(t *testing.T) {
	want := &DeviceDescription{
		Type:              "a",
		Address:           "b",
		RFAddress:         1,
		Children:          []string{"c", "d"},
		Parent:            "e",
		ParentType:        "f",
		Index:             2,
		AESActive:         3,
		Paramsets:         []string{"g"},
		Firmware:          "h",
		AvailableFirmware: "i",
		Version:           4,
		Flags:             5,
		LinkSourceRoles:   "j",
		LinkTargetRoles:   "k",
		Direction:         6,
		Group:             "l",
		Team:              "m",
		TeamTag:           "n",
		TeamChannels:      []string{"o", "p", "q"},
		Interface:         "r",
		Roaming:           7,
		RXMode:            8,
	}
	q := xmlrpc.Q(want.ToValue())
	got := &DeviceDescription{}
	got.ReadFrom(q)
	if q.Err() != nil {
		t.Fatal(q.Err())
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatal(got)
	}
}

func TestParameterDescription(t *testing.T) {
	cases := []*ParameterDescription{
		{
			Type:       "FLOAT",
			Operations: 1,
			Flags:      2,
			Default:    2.5,
			Max:        3.5,
			Min:        -1.5,
			Unit:       "a",
			TabOrder:   3,
			Control:    "b",
			ID:         "c",
			Special: []SpecialValue{
				{ID: "Zero", Value: 0.0},
				{ID: "One", Value: 1.0},
			},
		},
		{
			Type:       "INTEGER",
			Operations: 1,
			Flags:      2,
			Default:    2,
			Max:        3,
			Min:        -1,
			Unit:       "a",
			TabOrder:   3,
			Control:    "b",
			ID:         "c",
			Special: []SpecialValue{
				{ID: "Zero", Value: 0},
				{ID: "One", Value: 1},
			},
		},
		{
			Type:       "ENUM",
			Operations: 1,
			Flags:      2,
			Default:    1,
			Max:        0,
			Min:        2,
			Unit:       "a",
			TabOrder:   3,
			Control:    "b",
			ID:         "c",
			ValueList:  []string{"d", "e", "f"},
		},
	}
	for _, c := range cases {
		v, err := c.ToValue()
		if err != nil {
			t.Fatal(err)
		}
		q := xmlrpc.Q(v)
		got := &ParameterDescription{}
		got.ReadFrom(q)
		if q.Err() != nil {
			t.Fatal(q.Err())
		}
		if !reflect.DeepEqual(c, got) {
			t.Fatal(got)
		}
	}
}

func TestParamsetDescription(t *testing.T) {
	want := ParamsetDescription{
		"A": &ParameterDescription{
			Type:       "BOOL",
			Operations: 0,
			Flags:      1,
			Default:    true,
			Max:        true,
			Min:        false,
			Unit:       "",
			TabOrder:   2,
			Control:    "",
			ID:         "",
		},
		"B": &ParameterDescription{
			Type:       "STRING",
			Operations: 2,
			Flags:      3,
			Default:    "",
			Max:        "",
			Min:        "",
			Unit:       "",
			TabOrder:   4,
			Control:    "",
			ID:         "",
		},
	}
	v, err := want.ToValue()
	if err != nil {
		t.Fatal(err)
	}
	q := xmlrpc.Q(v)
	got := ParamsetDescription{}
	got.ReadFrom(q)
	if q.Err() != nil {
		t.Fatal(q.Err())
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatal(got)
	}
}
