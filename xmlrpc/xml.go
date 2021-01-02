package xmlrpc

import (
	"encoding/xml"
	"fmt"
	"github.com/mdzio/go-hmccu/model"
	"strconv"
)

// MethodCall represents an XML-RPC method call.
type MethodCall struct {
	MethodName string        `xml:"methodName"`
	Params     *model.Params `xml:"params"`
	XMLName    xml.Name      `xml:"methodCall"`
}

// MethodResponse represents an XML-RPC method response.
type MethodResponse struct {
	Params  *model.Params `xml:"params"`
	Fault   *model.Value  `xml:"fault>value"`
	XMLName xml.Name      `xml:"methodResponse"`
}

// MethodError encapsulates an XML-RPC fault response.
type MethodError struct {
	Code    int
	Message string
}

// Error implements the error interface.
func (f *MethodError) Error() string {
	return fmt.Sprintf("XML-RPC fault (code: %d, message: %s)", f.Code, f.Message)
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
		Fault: &model.Value{
			Struct: &model.Struct{
				[]*model.Member{
					{"faultCode", &model.Value{I4: strconv.Itoa(code)}},
					{"faultString", &model.Value{FlatString: message}},
				},
			},
		},
	}
}

func newMethodResponse(value *model.Value) *MethodResponse {
	return &MethodResponse{
		Params: &model.Params{
			[]*model.Param{
				&model.Param{value},
			},
		},
	}
}
