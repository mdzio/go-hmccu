package xmlrpc

import (
	"encoding/xml"
	"reflect"
	"testing"
)

type xmlTestCase struct {
	in   interface{}
	want string
}

func xmlRunMarshalTests(t *testing.T, cases []xmlTestCase) {
	for i, c := range cases {
		xml, err := xml.Marshal(c.in)
		if err != nil {
			t.Errorf("unexpected error in test case %d: %v", i+1, err)
		} else {
			xmltxt := string(xml)
			if xmltxt != c.want {
				t.Errorf("unexpected xml in test case %d: want: %s got: %s", i+1, c.want, xmltxt)
			}
		}
	}
}
func TestMarshalXMLValue(t *testing.T) {
	cases := []xmlTestCase{
		{
			// test case 1
			Value{I4: "123"},
			"<value><i4>123</i4></value>",
		},
		{
			// test case 2
			Value{Int: "0"},
			"<value><int>0</int></value>",
		},
		{
			// test case 3
			Value{Boolean: "1"},
			"<value><boolean>1</boolean></value>",
		},
		{
			// test case 4
			Value{ElemString: "abc"},
			"<value><string>abc</string></value>",
		},
		{
			// test case 5
			Value{FlatString: "def"},
			"<value>def</value>",
		},
		{
			// test case 6
			Value{Double: "123.456"},
			"<value><double>123.456</double></value>",
		},
		{
			// test case 7
			Value{DateTime: "2018-01-01T00:00:00"},
			"<value><dateTime.iso8601>2018-01-01T00:00:00</dateTime.iso8601></value>",
		},
		{
			// test case 8
			Value{Base64: "SGVsbG8gV29ybGQh"},
			"<value><base64>SGVsbG8gV29ybGQh</base64></value>",
		},
		{
			// test case 9
			Value{
				Struct: &Struct{
					Members: []*Member{},
				},
			},
			"<value><struct></struct></value>",
		},
		{
			// test case 10
			Value{
				Struct: &Struct{
					Members: []*Member{
						{"Field1", &Value{Int: "123"}},
						{"Field2", &Value{ElemString: "abc"}},
					},
				},
			},
			"<value><struct><member><name>Field1</name><value><int>123</int></value></member><member><name>Field2</name><value><string>abc</string></value></member></struct></value>",
		},
		{
			// test case 11
			Value{
				Array: &Array{},
			},
			"<value><array><data></data></array></value>",
		},
		{
			// test case 12
			Value{
				Array: &Array{
					[]*Value{
						{FlatString: "abc"},
						{I4: "4"},
					},
				},
			},
			"<value><array><data><value>abc</value><value><i4>4</i4></value></data></array></value>",
		},
		{
			// test case 13
			Value{
				Array: &Array{
					[]*Value{
						{I4: "4"},
						{
							Struct: &Struct{
								Members: []*Member{
									{"Field", &Value{FlatString: "abc"}},
								},
							},
						},
					},
				},
			},
			"<value><array><data><value><i4>4</i4></value><value><struct><member><name>Field</name><value>abc</value></member></struct></value></data></array></value>",
		},
	}
	xmlRunMarshalTests(t, cases)
}
func TestMarshal(t *testing.T) {
	cases := []xmlTestCase{
		{
			// test case 1
			MethodCall{
				MethodName: "noParameters",
				Params:     &Params{},
			},
			"<methodCall><methodName>noParameters</methodName><params></params></methodCall>",
		},
		{
			// test case 2
			MethodCall{
				MethodName: "setAnswer",
				Params: &Params{
					[]*Param{
						{&Value{I4: "42"}},
					},
				},
			},
			"<methodCall><methodName>setAnswer</methodName><params><param><value><i4>42</i4></value></param></params></methodCall>",
		},
		{
			// test case 3
			MethodCall{
				MethodName: "twoParameters",
				Params: &Params{
					[]*Param{
						{&Value{Boolean: "1"}},
						{&Value{ElemString: "abc"}},
					},
				},
			},
			"<methodCall><methodName>twoParameters</methodName><params><param><value><boolean>1</boolean></value></param><param><value><string>abc</string></value></param></params></methodCall>",
		},
		{
			// test case 4
			// example from spec.
			MethodResponse{
				Fault: &Value{
					Struct: &Struct{
						Members: []*Member{
							{
								"faultCode",
								&Value{Int: "4"},
							},
							{
								"faultString",
								&Value{ElemString: "Too many parameters."},
							},
						},
					},
				},
			},
			"<methodResponse><fault><value><struct><member><name>faultCode</name><value><int>4</int></value></member><member><name>faultString</name><value><string>Too many parameters.</string></value></member></struct></value></fault></methodResponse>",
		},
		{
			// test case 5
			// original CCU response: data type string without tags. i4 tag instead of int tag.
			MethodResponse{
				Fault: &Value{
					Struct: &Struct{
						Members: []*Member{
							{
								"faultCode",
								&Value{I4: "-1"},
							},
							{
								"faultString",
								&Value{FlatString: ": unknown method name"},
							},
						},
					},
				},
			},
			"<methodResponse><fault><value><struct><member><name>faultCode</name><value><i4>-1</i4></value></member><member><name>faultString</name><value>: unknown method name</value></member></struct></value></fault></methodResponse>",
		},
		{
			// test case 6
			// original CCU request.
			MethodCall{
				MethodName: "getDeviceDescription",
				Params: &Params{
					[]*Param{
						{&Value{FlatString: "GEQ0123456:1"}},
					},
				},
			},
			"<methodCall><methodName>getDeviceDescription</methodName><params><param><value>GEQ0123456:1</value></param></params></methodCall>",
		},
		{
			// test case 7
			// original CCU response.
			MethodResponse{
				Params: &Params{
					[]*Param{
						{
							&Value{
								Struct: &Struct{
									Members: []*Member{
										{
											"ADDRESS",
											&Value{FlatString: "GEQ0123456:1"},
										},
										{
											"AES_ACTIVE",
											&Value{I4: "1"},
										},
										{
											"DIRECTION",
											&Value{I4: "1"},
										},
										{
											"FLAGS",
											&Value{I4: "1"},
										},
										{
											"INDEX",
											&Value{I4: "1"},
										},
										{
											"LINK_SOURCE_ROLES",
											&Value{FlatString: "KEYMATIC SWITCH WINMATIC"},
										},
										{
											"LINK_TARGET_ROLES",
											&Value{FlatString: ""},
										},
										{
											"PARAMSETS",
											&Value{
												Array: &Array{
													[]*Value{
														{FlatString: "LINK"},
														{FlatString: "MASTER"},
														{FlatString: "VALUES"},
													},
												},
											},
										},
										{
											"PARENT",
											&Value{FlatString: "GEQ0123456"},
										},
										{
											"PARENT_TYPE",
											&Value{FlatString: "HM-Sec-MDIR"},
										},
										{
											"TYPE",
											&Value{FlatString: "MOTION_DETECTOR"},
										},
										{
											"VERSION",
											&Value{I4: "10"},
										},
									},
								},
							},
						},
					},
				},
			},
			"<methodResponse><params><param><value><struct><member><name>ADDRESS</name><value>GEQ0123456:1</value></member><member><name>AES_ACTIVE</name><value><i4>1</i4></value></member><member><name>DIRECTION</name><value><i4>1</i4></value></member><member><name>FLAGS</name><value><i4>1</i4></value></member><member><name>INDEX</name><value><i4>1</i4></value></member><member><name>LINK_SOURCE_ROLES</name><value>KEYMATIC SWITCH WINMATIC</value></member><member><name>LINK_TARGET_ROLES</name><value></value></member><member><name>PARAMSETS</name><value><array><data><value>LINK</value><value>MASTER</value><value>VALUES</value></data></array></value></member><member><name>PARENT</name><value>GEQ0123456</value></member><member><name>PARENT_TYPE</name><value>HM-Sec-MDIR</value></member><member><name>TYPE</name><value>MOTION_DETECTOR</value></member><member><name>VERSION</name><value><i4>10</i4></value></member></struct></value></param></params></methodResponse>",
		},
	}
	xmlRunMarshalTests(t, cases)
}

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
		{Value{ElemString: "abc"}, "abc"},
		{Value{FlatString: " def"}, " def"},
		{Value{ElemString: "abc", FlatString: "def"}, "abc"},
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
					{"name2", &Value{ElemString: "abc"}},
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
					{"name2", &Value{ElemString: "abc"}},
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
					{FlatString: "abc"},
					{I4: "4"},
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
					{FlatString: "abc"},
					{ElemString: "def"},
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
			&Value{Array: &Array{[]*Value{{FlatString: "abc"}}}},
			[]string{"abc"},
		},
		{
			&Value{Array: &Array{[]*Value{{Double: "123.456"}}}},
			[]interface{}{123.456},
		},
		{
			&Value{Struct: &Struct{[]*Member{{"abc", &Value{I4: "123"}}}}},
			map[string]interface{}{"abc": 123},
		},
		{
			&Value{Struct: &Struct{[]*Member{{
				"k",
				&Value{Array: &Array{[]*Value{{FlatString: "a"}, {FlatString: "b"}}}},
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
