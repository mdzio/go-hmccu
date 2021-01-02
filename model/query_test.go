package model

import (
	"reflect"
	"testing"
)

func TestQuery_Int(t *testing.T) {
	cases := []struct {
		in        Value
		wanted    int
		errWanted bool
	}{
		{Value{}, 0, true},
		{Value{I4: ""}, 0, true},
		{Value{I4: "123"}, 123, false},
		{Value{Int: "456"}, 456, false},
	}
	for _, c := range cases {
		e := Q(&c.in)
		i := e.Int()
		err := e.Err()
		if i != c.wanted || (err != nil) != c.errWanted {
			t.Fail()
		}
	}
}

func TestQuery_Boolean(t *testing.T) {
	cases := []struct {
		in        Value
		wanted    bool
		errWanted bool
	}{
		{Value{}, false, true},
		{Value{Boolean: "2"}, false, true},
		{Value{Boolean: "0"}, false, false},
		{Value{Boolean: "1"}, true, false},
	}
	for _, c := range cases {
		u := Q(&c.in)
		b := u.Bool()
		err := u.Err()
		if b != c.wanted || (err != nil) != c.errWanted {
			t.Fail()
		}
	}
}

func TestQuery_String(t *testing.T) {
	cases := []struct {
		in     Value
		wanted string
	}{
		{Value{String: "abc"}, "abc"},
		{Value{FlatString: " def"}, " def"},
		{Value{String: "abc", FlatString: "def"}, "abc"},
	}
	for _, c := range cases {
		u := Q(&c.in)
		s := u.String()
		if s != c.wanted {
			t.Fail()
		}
	}
}

func TestQuery_Double(t *testing.T) {
	cases := []struct {
		in        Value
		wanted    float64
		errWanted bool
	}{
		{Value{}, 0.0, true},
		{Value{Double: "a"}, 0.0, true},
		{Value{Double: "0"}, 0.0, false},
		{Value{Double: "1"}, 1.0, false},
		{Value{Double: "-1e3"}, -1000.0, false},
	}
	for _, c := range cases {
		u := Q(&c.in)
		d := u.Float64()
		err := u.Err()
		if d != c.wanted || (err != nil) != c.errWanted {
			t.Fail()
		}
	}
}

func TestQuery_Key(t *testing.T) {
	e := Q(&Value{Struct: &Struct{}})
	e.Key("unknown")
	err := e.Err()
	if err == nil {
		t.Fail()
	}

	e = Q(
		&Value{
			Struct: &Struct{
				Members: []*Member{
					{"name1", &Value{I4: "123"}},
					{"name2", &Value{String: "abc"}},
				},
			},
		},
	)

	e.Key("unknown")
	err = e.Err()
	if err == nil {
		t.Fail()
	}
	*e.err = nil

	f := e.Key("name1")
	err = e.Err()
	if err != nil {
		t.Fail()
	}
	i := f.Int()
	err = f.Err()
	if err != nil || i != 123 {
		t.Fail()
	}

	i = e.Key("name1").Int()
	err = e.Err()
	if err != nil || i != 123 {
		t.Fail()
	}

	s := e.Key("name2").String()
	err = e.Err()
	if err != nil || s != "abc" {
		t.Fail()
	}

	s = e.Key("name2").Key("unknown").Key("unknown2").String()
	err = e.Err()
	if err == nil || s != "" {
		t.Fail()
	}
}

func TestQuery_TryKey(t *testing.T) {
	e := Q(
		&Value{
			Struct: &Struct{
				Members: []*Member{
					{"name1", &Value{I4: "123"}},
					{"name2", &Value{String: "abc"}},
				},
			},
		},
	)
	i := e.TryKey("name1").Int()
	if i != 123 || e.Err() != nil {
		t.Fail()
	}
	i = e.TryKey("unknown").Int()
	if i != 0 || e.Err() != nil {
		t.Fail()
	}
	i = e.TryKey("name1").TryKey("unkown").Int()
	if i != 0 || e.Err() == nil {
		t.Fail()
	}
}

func TestQuery_Array(t *testing.T) {
	e := Q(
		&Value{
			Array: &Array{
				[]*Value{
					&Value{FlatString: "abc"},
					&Value{I4: "4"},
				},
			},
		},
	)
	if len(e.Slice()) != 2 {
		t.Fail()
	}
	s := e.Slice()[0].String()
	i := e.Slice()[1].Int()
	if s != "abc" || i != 4 || e.Err() != nil {
		t.Fail()
	}
	e.Slice()[0].Int()
	if e.Err() == nil {
		t.Fail()
	}
	*e.err = nil

	e = Q(&Value{Double: "123.456"})
	e.Slice()
	if e.Err() == nil {
		t.Fail()
	}
}

func TestQuery_Strings(t *testing.T) {
	e := Q(
		&Value{
			Array: &Array{
				[]*Value{
					&Value{FlatString: "abc"},
					&Value{String: "def"},
				},
			},
		},
	)
	s := e.Strings()
	if e.Err() != nil {
		t.Error(e.Err())
	}
	if !reflect.DeepEqual(s, []string{"abc", "def"}) {
		t.Error("invalid result: ", s)
	}
}

func TestQuery_Any(t *testing.T) {
	cases := []struct {
		v       *Value
		want    interface{}
		wantErr bool
	}{
		{&Value{I4: "123"}, int(123), false},
		{&Value{Boolean: "1"}, true, false},
		{&Value{Double: "123.456"}, 123.456, false},
		{&Value{FlatString: "abc"}, "abc", false},
		{&Value{Double: "a"}, 0, true},
		{nil, nil, false},
	}
	for _, c := range cases {
		e := Q(c.v)
		v := e.Any()
		if (e.Err() != nil) && !c.wantErr {
			t.Errorf("unexpected error: %v", e.Err())
		} else if (e.Err() == nil) && c.wantErr {
			t.Error("missing error")
		}
		if e.Err() == nil && !reflect.DeepEqual(v, c.want) {
			t.Errorf("unexpected value: %v, expected: %v", v, c.want)
		}
	}
}
