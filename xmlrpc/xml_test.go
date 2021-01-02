package xmlrpc

import (
	"encoding/xml"
	"github.com/mdzio/go-hmccu/model"
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
			model.Value{I4: "123"},
			"<value><i4>123</i4></value>",
		},
		{
			// test case 2
			model.Value{Int: "0"},
			"<value><int>0</int></value>",
		},
		{
			// test case 3
			model.Value{Boolean: "1"},
			"<value><boolean>1</boolean></value>",
		},
		{
			// test case 4
			model.Value{String: "abc"},
			"<value><string>abc</string></value>",
		},
		{
			// test case 5
			model.Value{FlatString: "def"},
			"<value>def</value>",
		},
		{
			// test case 6
			model.Value{Double: "123.456"},
			"<value><double>123.456</double></value>",
		},
		{
			// test case 7
			model.Value{DateTime: "2018-01-01T00:00:00"},
			"<value><dateTime.iso8601>2018-01-01T00:00:00</dateTime.iso8601></value>",
		},
		{
			// test case 8
			model.Value{Base64: "SGVsbG8gV29ybGQh"},
			"<value><base64>SGVsbG8gV29ybGQh</base64></value>",
		},
		{
			// test case 9
			model.Value{
				Struct: &model.Struct{
					Members: []*model.Member{},
				},
			},
			"<value><struct></struct></value>",
		},
		{
			// test case 10
			model.Value{
				Struct: &model.Struct{
					Members: []*model.Member{
						{"Field1", &model.Value{Int: "123"}},
						{"Field2", &model.Value{String: "abc"}},
					},
				},
			},
			"<value><struct><member><name>Field1</name><value><int>123</int></value></member><member><name>Field2</name><value><string>abc</string></value></member></struct></value>",
		},
		{
			// test case 11
			model.Value{
				Array: &model.Array{},
			},
			"<value><array><data></data></array></value>",
		},
		{
			// test case 12
			model.Value{
				Array: &model.Array{
					[]*model.Value{
						&model.Value{FlatString: "abc"},
						&model.Value{I4: "4"},
					},
				},
			},
			"<value><array><data><value>abc</value><value><i4>4</i4></value></data></array></value>",
		},
		{
			// test case 13
			model.Value{
				Array: &model.Array{
					[]*model.Value{
						&model.Value{I4: "4"},
						&model.Value{
							Struct: &model.Struct{
								Members: []*model.Member{
									{"Field", &model.Value{FlatString: "abc"}},
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
				Params:     &model.Params{},
			},
			"<methodCall><methodName>noParameters</methodName><params></params></methodCall>",
		},
		{
			// test case 2
			MethodCall{
				MethodName: "setAnswer",
				Params: &model.Params{
					[]*model.Param{
						{&model.Value{I4: "42"}},
					},
				},
			},
			"<methodCall><methodName>setAnswer</methodName><params><param><value><i4>42</i4></value></param></params></methodCall>",
		},
		{
			// test case 3
			MethodCall{
				MethodName: "twoParameters",
				Params: &model.Params{
					[]*model.Param{
						{&model.Value{Boolean: "1"}},
						{&model.Value{String: "abc"}},
					},
				},
			},
			"<methodCall><methodName>twoParameters</methodName><params><param><value><boolean>1</boolean></value></param><param><value><string>abc</string></value></param></params></methodCall>",
		},
		{
			// test case 4
			// example from spec.
			MethodResponse{
				Fault: &model.Value{
					Struct: &model.Struct{
						Members: []*model.Member{
							{
								"faultCode",
								&model.Value{Int: "4"},
							},
							{
								"faultString",
								&model.Value{String: "Too many parameters."},
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
				Fault: &model.Value{
					Struct: &model.Struct{
						Members: []*model.Member{
							{
								"faultCode",
								&model.Value{I4: "-1"},
							},
							{
								"faultString",
								&model.Value{FlatString: ": unknown method name"},
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
				Params: &model.Params{
					[]*model.Param{
						{&model.Value{FlatString: "GEQ0123456:1"}},
					},
				},
			},
			"<methodCall><methodName>getDeviceDescription</methodName><params><param><value>GEQ0123456:1</value></param></params></methodCall>",
		},
		{
			// test case 7
			// original CCU response.
			MethodResponse{
				Params: &model.Params{
					[]*model.Param{
						{
							&model.Value{
								Struct: &model.Struct{
									Members: []*model.Member{
										{
											"ADDRESS",
											&model.Value{FlatString: "GEQ0123456:1"},
										},
										{
											"AES_ACTIVE",
											&model.Value{I4: "1"},
										},
										{
											"DIRECTION",
											&model.Value{I4: "1"},
										},
										{
											"FLAGS",
											&model.Value{I4: "1"},
										},
										{
											"INDEX",
											&model.Value{I4: "1"},
										},
										{
											"LINK_SOURCE_ROLES",
											&model.Value{FlatString: "KEYMATIC SWITCH WINMATIC"},
										},
										{
											"LINK_TARGET_ROLES",
											&model.Value{FlatString: ""},
										},
										{
											"PARAMSETS",
											&model.Value{
												Array: &model.Array{
													[]*model.Value{
														&model.Value{FlatString: "LINK"},
														&model.Value{FlatString: "MASTER"},
														&model.Value{FlatString: "VALUES"},
													},
												},
											},
										},
										{
											"PARENT",
											&model.Value{FlatString: "GEQ0123456"},
										},
										{
											"PARENT_TYPE",
											&model.Value{FlatString: "HM-Sec-MDIR"},
										},
										{
											"TYPE",
											&model.Value{FlatString: "MOTION_DETECTOR"},
										},
										{
											"VERSION",
											&model.Value{I4: "10"},
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
