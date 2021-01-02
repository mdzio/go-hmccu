package model

import "encoding/xml"

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
	String     string   `xml:"string,omitempty"`
	FlatString string   `xml:",chardata"`
	Double     string   `xml:"double,omitempty"`
	DateTime   string   `xml:"dateTime.iso8601,omitempty"`
	Base64     string   `xml:"base64,omitempty"`
	Struct     *Struct  `xml:"struct"`
	Array      *Array   `xml:"array"`
	XMLName    xml.Name `xml:"value"`
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
