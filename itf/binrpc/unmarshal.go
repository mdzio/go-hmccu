package binrpc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"strconv"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"golang.org/x/text/encoding/charmap"
)

// Decoder decodes BIN-RPC requests.
type Decoder struct {
	r io.Reader
}

// NewDecoder create a Decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// DecodeRequest decodes an BIN-RPC request.
func (d *Decoder) DecodeRequest() (string, xmlrpc.Values, error) {
	// read header
	var hdr header
	if err := binary.Read(d.r, binary.BigEndian, &hdr); err != nil {
		return "", nil, fmt.Errorf("Reading of header failed: %w", err)
	}

	// check marker and message type
	if hdr.Marker != binrpcMarker {
		return "", nil, fmt.Errorf("Invalid start of header: %sh", hex.EncodeToString(hdr.Marker[:]))
	}
	if hdr.MsgType != msgTypeRequest {
		return "", nil, fmt.Errorf("Invalid message type: %Xh", hdr.MsgType)
	}

	// read method name
	method, err := d.decodeString()
	if err != nil {
		return "", nil, fmt.Errorf("Reading of method name failed: %w", err)
	}

	// read parameters
	params, err := d.decodeValues()
	return string(method.FlatString), params, err
}

// DecodeResponse decodes a BIN-RPC response/fault. A received fault packet is
// returned as xmlrpc.MethodError.
func (d *Decoder) DecodeResponse() (*xmlrpc.Value, error) {
	// read hdr
	var hdr header
	if err := binary.Read(d.r, binary.BigEndian, &hdr); err != nil {
		return nil, fmt.Errorf("Reading of header failed: %w", err)
	}

	// check marker
	if hdr.Marker != binrpcMarker {
		return nil, fmt.Errorf("Invalid start of header: %s", hex.EncodeToString(hdr.Marker[:]))
	}

	// message type?
	switch hdr.MsgType {

	case msgTypeResponse:
		// valid response
		return d.decodeValue()

	case msgTypeFault:
		// fault response
		v, err := d.decodeValue()
		if err != nil {
			return nil, fmt.Errorf("Decoding of fault response failed: %w", err)
		}
		f := xmlrpc.Q(v)
		code := f.Key("faultCode").Int()
		msg := f.Key("faultString").String()
		if f.Err() != nil {
			return nil, fmt.Errorf("Invalid fault response: %w", f.Err())
		}
		// return fault as error
		return nil, &xmlrpc.MethodError{Code: code, Message: msg}
	}
	return nil, fmt.Errorf("Unexpected message type: %02Xh", hdr.MsgType)
}

func (d *Decoder) decodeValues() (xmlrpc.Values, error) {
	// read length
	var length uint32
	if err := binary.Read(d.r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("Reading of length failed: %w", err)
	}

	// read items
	vals := make([]*xmlrpc.Value, length)
	for i := range vals {
		val, err := d.decodeValue()
		if err != nil {
			return nil, err
		}
		vals[i] = val
	}
	return vals, nil
}

func (d *Decoder) decodeValue() (*xmlrpc.Value, error) {
	// read data type
	var valueType uint32
	if err := binary.Read(d.r, binary.BigEndian, &valueType); err != nil {
		return nil, fmt.Errorf("Reading of data type failed: %w", err)
	}

	// read value
	switch valueType {
	case integerType:
		return d.decodeInteger()
	case booleanType:
		return d.decodeBool()
	case stringType:
		return d.decodeString()
	case doubleType:
		return d.decodeDouble()
	case arrayType:
		return d.decodeArray()
	case structType:
		return d.decodeStruct()
	}
	return nil, fmt.Errorf("Unkwon value type: %Xh", valueType)
}

func (d *Decoder) decodeString() (*xmlrpc.Value, error) {
	// read string length
	var length uint32
	if err := binary.Read(d.r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("Reading of string length failed: %w", err)
	}

	// read ISO8859-1 string
	bISO8859_1 := make([]byte, int(length))
	if err := binary.Read(d.r, binary.BigEndian, &bISO8859_1); err != nil {
		return nil, fmt.Errorf("Reading of string content failed: %w", err)
	}

	// decode ISO8859-1 to UTF8
	rUTF8 := charmap.ISO8859_1.NewDecoder().Reader(bytes.NewBuffer(bISO8859_1))
	bUTF8, err := ioutil.ReadAll(rUTF8)
	if err != nil {
		return nil, fmt.Errorf("Converting of string content failed: %w", err)
	}
	return &xmlrpc.Value{FlatString: string(bUTF8)}, nil
}

func (d *Decoder) decodeInteger() (*xmlrpc.Value, error) {
	var val int32
	if err := binary.Read(d.r, binary.BigEndian, &val); err != nil {
		return nil, fmt.Errorf("Reading of integer failed: %w", err)
	}
	return &xmlrpc.Value{I4: strconv.Itoa(int(val))}, nil
}

func (d *Decoder) decodeBool() (*xmlrpc.Value, error) {
	var val uint8
	if err := binary.Read(d.r, binary.BigEndian, &val); err != nil {
		return nil, fmt.Errorf("Reading of bool failed: %w", err)
	}
	if val != 0 {
		return &xmlrpc.Value{Boolean: "1"}, nil
	}
	return &xmlrpc.Value{Boolean: "0"}, nil
}

func (d *Decoder) decodeDouble() (*xmlrpc.Value, error) {
	// read mantissa and exponent
	var double struct {
		Man int32
		Exp int32
	}
	if err := binary.Read(d.r, binary.BigEndian, &double); err != nil {
		return nil, fmt.Errorf("Reading of double failed: %w", err)
	}

	// convert
	val := math.Pow(2, float64(double.Exp)) * float64(double.Man) / mantissaMultiplicator
	return &xmlrpc.Value{Double: strconv.FormatFloat(val, 'f', -1, 64)}, nil
}

func (d *Decoder) decodeArray() (*xmlrpc.Value, error) {
	vals, err := d.decodeValues()
	if err != nil {
		return nil, err
	}
	return &xmlrpc.Value{Array: &xmlrpc.Array{Data: vals}}, nil
}

func (d *Decoder) decodeStruct() (*xmlrpc.Value, error) {
	var length uint32
	if err := binary.Read(d.r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("Failed to decode struct length: %w", err)
	}

	val := &xmlrpc.Value{
		Struct: &xmlrpc.Struct{Members: []*xmlrpc.Member{}},
	}

	for i := 0; i < int(length); i++ {
		keyVal, err := d.decodeString()
		if err != nil {
			return nil, fmt.Errorf("Failed to decode stuct key: %w", err)
		}

		structVal, err := d.decodeValue()
		if err != nil {
			return nil, fmt.Errorf("Failed to decode struct value: %w", err)
		}
		val.Struct.Members = append(val.Struct.Members, &xmlrpc.Member{
			Name:  keyVal.FlatString,
			Value: structVal,
		})
	}

	return val, nil
}
