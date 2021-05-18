package xmlrpc

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// MethodCall represents an XML-RPC method call.
type MethodCall struct {
	MethodName string   `xml:"methodName"`
	Params     *Params  `xml:"params"`
	XMLName    xml.Name `xml:"methodCall"`
}

// MethodResponse represents an XML-RPC method response.
type MethodResponse struct {
	Params  *Params  `xml:"params"`
	Fault   *Value   `xml:"fault>value"`
	XMLName xml.Name `xml:"methodResponse"`
}

// Params holds the parameters for the method call or response.
type Params struct {
	Param []*Param `xml:"param"`
}

// Param is a single parameter.
type Param struct {
	Value *Value
}

// Value represents an XML-RPC value.
type Value struct {
	I4         string   `xml:"i4,omitempty"`
	Int        string   `xml:"int,omitempty"`
	Boolean    string   `xml:"boolean,omitempty"`
	ElemString string   `xml:"string,omitempty"`
	FlatString string   `xml:",chardata"`
	Double     string   `xml:"double,omitempty"`
	DateTime   string   `xml:"dateTime.iso8601,omitempty"`
	Base64     string   `xml:"base64,omitempty"`
	Struct     *Struct  `xml:"struct"`
	Array      *Array   `xml:"array"`
	XMLName    xml.Name `xml:"value"`
}

// String implements the Stringer interface. Data types are indicated by the
// representation.
func (v *Value) String() string {
	if v.I4 != "" {
		return v.I4
	}
	if v.Int != "" {
		return v.Int
	}
	if v.Boolean != "" {
		switch v.Boolean {
		case "0":
			return "false"
		case "1":
			return "true"
		default:
			return v.Boolean
		}
	}
	if v.Double != "" {
		if strings.ContainsRune(v.Double, '.') {
			return v.Double
		}
		return v.Double + ".0"
	}
	if v.DateTime != "" {
		return v.DateTime + "(time)"
	}
	if v.Base64 != "" {
		return v.Base64 + "(base64)"
	}
	if v.ElemString != "" {
		return strconv.Quote(v.ElemString)
	}
	if v.Array != nil {
		var sb strings.Builder
		sb.WriteRune('[')
		first := true
		for i := range v.Array.Data {
			if first {
				first = false
			} else {
				sb.WriteRune(' ')
			}
			sb.WriteString(v.Array.Data[i].String())
		}
		sb.WriteRune(']')
		return sb.String()
	}
	if v.Struct != nil {
		var sb strings.Builder
		sb.WriteRune('{')
		first := true
		for i := range v.Struct.Members {
			if first {
				first = false
			} else {
				sb.WriteRune(' ')
			}
			sb.WriteString(v.Struct.Members[i].Name)
			sb.WriteRune(':')
			sb.WriteString(v.Struct.Members[i].Value.String())
		}
		sb.WriteRune('}')
		return sb.String()
	}
	return strconv.Quote(v.FlatString)
}

// Values is a slice of Value's. The type is used by the Caller interface.
type Values []*Value

// String implements the Stringer interface.
func (vs Values) String() string {
	var sb strings.Builder
	first := true
	for i := range vs {
		if first {
			first = false
		} else {
			sb.WriteRune(' ')
		}
		sb.WriteString(vs[i].String())
	}
	return sb.String()
}

// Struct represents an XML-RPC struct.
type Struct struct {
	Members []*Member `xml:"member"`
}

// Member represents an XML-RPC struct member.
type Member struct {
	Name  string `xml:"name"`
	Value *Value
}

// Array represents an XML-RPC array.
type Array struct {
	Data []*Value `xml:"data>value"`
}

// MethodError encapsulates an XML-RPC fault response.
type MethodError struct {
	Code    int
	Message string
}

// Error implements the error interface.
func (f *MethodError) Error() string {
	return fmt.Sprintf("RPC fault (code: %d, message: %s)", f.Code, f.Message)
}

// Query helps to extract values from the XML model.
type Query struct {
	value *Value
	err   *error
	// faster lookup for structs
	lookup map[string]*Query
	// cache arrays
	array []*Query
}

// Q creates a new Query for the specified Value.
func Q(v *Value) *Query {
	var err error
	return &Query{value: v, err: &err}
}

// Err returns the first encountered error.
func (q *Query) Err() error {
	return *q.err
}

// Int gets an XML-RPC int or i4 value.
func (q *Query) Int() (i int) {
	// previous error or empty optional?
	if q.Err() != nil || q.value == nil {
		return
	}
	var s string
	if q.value.I4 != "" {
		s = q.value.I4
	} else if q.value.Int != "" {
		s = q.value.Int
	} else {
		*q.err = errors.New("Not an int")
		return
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		*q.err = fmt.Errorf("Invalid int: %s", s)
		return
	}
	return
}

// Bool gets an XML-RPC boolean value.
func (q *Query) Bool() bool {
	// previous error or empty optional?
	if q.Err() != nil || q.value == nil {
		return false
	}
	switch q.value.Boolean {
	case "0":
		return false
	case "1":
		return true
	default:
		*q.err = errors.New("Not a bool or invalid")
		return false
	}
}

// String gets an XML-RPC string value.
func (q *Query) String() string {
	// previous error or empty optional?
	if q.Err() != nil || q.value == nil {
		return ""
	}
	// first string variant
	if q.value.ElemString != "" {
		return q.value.ElemString
	}
	// exclude other types
	if q.value.Boolean != "" || q.value.I4 != "" || q.value.Int != "" || q.value.Double != "" ||
		q.value.Base64 != "" || q.value.DateTime != "" || q.value.Array != nil || q.value.Struct != nil {
		*q.err = errors.New("Not a string")
	}
	// second string variant
	return q.value.FlatString
}

func (q *Query) allZero() bool {
	return q.value.Boolean == "" && q.value.I4 == "" && q.value.Int == "" && q.value.Double == "" &&
		q.value.ElemString == "" && q.value.FlatString == "" && q.value.Base64 == "" &&
		q.value.DateTime == "" && q.value.Array == nil && q.value.Struct == nil
}

// IsEmpty returns true, if there is no previous error and the value is empty.
// An empty value can also be interpreted as an empty string.
func (q *Query) IsEmpty() bool {
	// previous error?
	if q.Err() != nil {
		return false
	}
	// empty optional?
	if q.value == nil {
		return true
	}
	// all fields have zero value?
	return q.allZero()
}

// IsNotEmpty returns true, if there is no previous error and the value is not
// empty. An empty value can also be interpreted as an empty string.
func (q *Query) IsNotEmpty() bool {
	// previous error or empty optional?
	if q.Err() != nil || q.value == nil {
		return false
	}
	// any field has not zero value?
	return !q.allZero()
}

// Float64 gets an XML-RPC double value.
func (q *Query) Float64() float64 {
	// previous error or empty optional?
	if q.Err() != nil || q.value == nil {
		return 0
	}
	if q.value.Double == "" {
		*q.err = errors.New("Not a double")
		return 0
	}
	d, err := strconv.ParseFloat(q.value.Double, 64)
	if err != nil {
		*q.err = fmt.Errorf("Invalid double: %s", q.value.Double)
		return 0
	}
	return d
}

// Any returns data type int, bool, float64, string or nil for an empty
// optional. For Struct or Array an error is set.
func (q *Query) Any() interface{} {
	// previous error or empty optional?
	if q.Err() != nil || q.value == nil {
		return nil
	}
	// detect data type
	if q.value.I4 != "" || q.value.Int != "" {
		return q.Int()
	} else if q.value.Boolean != "" {
		return q.Bool()
	} else if q.value.Double != "" {
		return q.Float64()
	} else if q.value.Struct != nil {
		*q.err = errors.New("Unexpected struct")
		return nil
	} else if q.value.Array != nil {
		*q.err = errors.New("Unexpected array")
		return nil
	}
	return q.String()
}

// Map returns all members of an XML-RPC struct.
func (q *Query) Map() map[string]*Query {
	// previous error or empty optional?
	if q.Err() != nil || q.value == nil {
		// return empty map
		return nil
	}
	// is map already created?
	if q.lookup != nil {
		return q.lookup
	}
	// create map
	s := q.value.Struct
	if s == nil {
		*q.err = errors.New("Not a struct")
		return nil
	}
	q.lookup = make(map[string]*Query)
	for _, m := range s.Members {
		q.lookup[m.Name] = &Query{value: m.Value, err: q.err}
	}
	return q.lookup
}

// key gets the specified member from a struct.
func (q *Query) key(name string, must bool) *Query {
	m := q.Map()
	// previous error?
	if q.Err() != nil {
		return &Query{err: q.err}
	}
	// lookup
	f, ok := m[name]
	if !ok {
		if must {
			*q.err = fmt.Errorf("Field not found: %s", name)
		}
		return &Query{err: q.err}
	}
	return f
}

// Key sets an error, if the specified member is missing.
func (q *Query) Key(name string) *Query {
	return q.key(name, true)
}

// TryKey does not set an error, if the specified member is missing.
func (q *Query) TryKey(name string) *Query {
	return q.key(name, false)
}

// Slice returns all array elements.
func (q *Query) Slice() []*Query {
	// previous error or empty optional?
	if q.Err() != nil || q.value == nil {
		// return empty slice
		return nil
	}
	// array already created?
	if q.array != nil {
		return q.array
	}
	// create array
	a := q.value.Array
	if a == nil {
		*q.err = fmt.Errorf("Not an array")
		return nil
	}
	q.array = make([]*Query, len(a.Data))
	for i, v := range a.Data {
		q.array[i] = &Query{value: v, err: q.err}
	}
	return q.array
}

// Strings returns a string array.
func (q *Query) Strings() []string {
	// previous error or empty optional?
	if q.Err() != nil || q.value == nil {
		// return empty slice
		return nil
	}
	// create array
	var r []string
	s := q.Slice()
	for _, e := range s {
		r = append(r, e.String())
	}
	if q.Err() != nil {
		// return empty slice
		return nil
	}
	return r
}

// Idx returns the array element at i.
func (q *Query) Idx(i int) *Query {
	s := q.Slice()
	// previous error
	if q.Err() != nil {
		return &Query{err: q.err}
	}
	// check bounds
	if i < 0 || i >= len(s) {
		*q.err = fmt.Errorf("Index out of bounds (array length: %d): %d", len(s), i)
		return &Query{err: q.err}
	}
	return s[i]
}

// Value returns the wrapped Value.
func (q *Query) Value() *Value {
	return q.value
}

// NewBool creates an xmlrpc.Value from a bool.
func NewBool(val bool) *Value {
	out := &Value{}
	if val {
		out.Boolean = "1"
	} else {
		out.Boolean = "0"
	}
	return out
}

// NewInt creates an xmlrpc.Value from an int.
func NewInt(val int) *Value {
	return &Value{I4: strconv.Itoa(val)}
}

// NewFloat64 creates an xmlrpc.Value from a float64.
func NewFloat64(val float64) *Value {
	return &Value{Double: strconv.FormatFloat(val, 'f', -1, 64)}
}

// NewString creates an xmlrpc.Value from a string.
func NewString(val string) *Value {
	return &Value{FlatString: val}
}

// NewStrings creates an xmlrpc.Value from a string slice.
func NewStrings(val []string) *Value {
	es := make([]*Value, len(val))
	for i := range val {
		es[i] = &Value{FlatString: val[i]}
	}
	return &Value{Array: &Array{es}}
}

// NewSlice creates an xmlrpc.Value from a slice.
func NewSlice(val []interface{}) (*Value, error) {
	es := make([]*Value, len(val))
	for i := range val {
		cv, err := NewValue(val[i])
		if err != nil {
			return nil, err
		}
		es[i] = cv
	}
	return &Value{Array: &Array{es}}, nil
}

// NewMap creates an xmlrpc.Value from a map.
func NewMap(val map[string]interface{}) (*Value, error) {
	ms := make([]*Member, len(val))
	i := 0
	for n, v := range val {
		cv, err := NewValue(v)
		if err != nil {
			return nil, err
		}
		ms[i] = &Member{Name: n, Value: cv}
		i++
	}
	return &Value{Struct: &Struct{Members: ms}}, nil
}

// NewValue creates a value from a native data type. Supported types: bool, int,
// float64, string, []string, []interface{} and map[string]interface{}.
func NewValue(in interface{}) (*Value, error) {
	switch val := in.(type) {
	case bool:
		return NewBool(val), nil
	case int:
		return NewInt(val), nil
	case float64:
		return NewFloat64(val), nil
	case string:
		return NewString(val), nil
	case []string:
		return NewStrings(val), nil
	case []interface{}:
		return NewSlice(val)
	case map[string]interface{}:
		return NewMap(val)
	default:
		return nil, fmt.Errorf("Conversion of type %[1]T with value %[1]v is not supported", in)
	}
}

func newFaultResponse(err error) *MethodResponse {
	var code int
	var message string
	if fre, ok := err.(*MethodError); ok {
		code = fre.Code
		message = fre.Message
	} else {
		code = -1
		message = err.Error()
	}
	return &MethodResponse{
		Fault: &Value{
			Struct: &Struct{
				[]*Member{
					{"faultCode", &Value{I4: strconv.Itoa(code)}},
					{"faultString", &Value{FlatString: message}},
				},
			},
		},
	}
}

func newMethodResponse(value *Value) *MethodResponse {
	return &MethodResponse{
		Params: &Params{
			[]*Param{{value}},
		},
	}
}
