package model

import (
	"reflect"
	"testing"
)

func TestNewValue(t *testing.T) {
	cases := []struct {
		want *Value
		in   interface{}
	}{
		{&Value{I4: "123"}, int(123)},
		{&Value{Boolean: "1"}, true},
		{&Value{Boolean: "0"}, false},
		{&Value{Double: "123.456"}, 123.456},
		{&Value{FlatString: "abc"}, "abc"},
		{
			&Value{Array: &Array{[]*Value{&Value{FlatString: "abc"}}}},
			[]string{"abc"},
		},
		{
			&Value{Array: &Array{[]*Value{&Value{Double: "123.456"}}}},
			[]interface{}{123.456},
		},
		{
			&Value{Struct: &Struct{[]*Member{&Member{"abc", &Value{I4: "123"}}}}},
			map[string]interface{}{"abc": 123},
		},
		{
			&Value{Struct: &Struct{[]*Member{&Member{
				"k",
				&Value{Array: &Array{[]*Value{&Value{FlatString: "a"}, &Value{FlatString: "b"}}}},
			}}}},
			map[string]interface{}{"k": []string{"a", "b"}},
		},
	}
	for _, c := range cases {
		v, err := NewValue(c.in)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(v, c.want) {
			t.Errorf("unexpected value: %v, expected: %v", v, c.want)
		}
	}
}
