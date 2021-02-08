package binrpc

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

// Decoder decodes BIN-RPC requests.
type Decoder struct {
	b *bufio.Reader
}

// NewDecoder create a Decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		b: bufio.NewReader(r),
	}
}

// DecodeRequest decodes an BIN-RPC request.
func (d *Decoder) DecodeRequest() (string, []*xmlrpc.Value, error) {
	var header struct {
		Head      [3]byte
		MsgType   uint8
		MsgSize   uint32
		MethodLen uint32
	}

	if err := binary.Read(d.b, binary.BigEndian, &header); err != nil {
		fmt.Printf("Failed to decode header: %s\n", err)
		return "", nil, fmt.Errorf("Failed to decode header")
	}

	method := make([]byte, int(header.MethodLen))
	if err := binary.Read(d.b, binary.BigEndian, &method); err != nil {
		fmt.Printf("Failed to decode method: %s\n", err)
		return "", nil, fmt.Errorf("Failed to decode method ")
	}

	params, err := d.decodeParams()
	return string(method), params, err
}

// DecodeResponseOrFault decodes a BIN-RPC response/fault.
func (d *Decoder) DecodeResponseOrFault() (*xmlrpc.Value, error) {
	// read header
	if err := binary.Read(d.b, binary.BigEndian, &header); err != nil {
		return nil, fmt.Errorf("Reading of header failed: %w", err)
	}
	// check marker
	if header.Marker != binrpcMarker {
		return nil, fmt.Errorf("Invalid start of header: %s", hex.EncodeToString(header.Marker[:]))
	}
	// message type?
	switch header.MsgType {
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
			return nil, fmt.Errorf("Invalid fault response: %v", f.Err())
		}
		return nil, &xmlrpc.MethodError{Code: code, Message: msg}
	}
	return nil, fmt.Errorf("Unexpected message type: %02Xh", header.MsgType)
}

func (d *Decoder) decodeParams() ([]*xmlrpc.Value, error) {
	var elementCount uint32
	if err := binary.Read(d.b, binary.BigEndian, &elementCount); err != nil {
		return nil, fmt.Errorf("Failed to decode element count ")
	}

	return d.decodeParamValues(elementCount)

}

func (d *Decoder) decodeParamValues(elementCount uint32) ([]*xmlrpc.Value, error) {
	vals := []*xmlrpc.Value{}
	for i := 0; i < int(elementCount); i++ {
		val, err := d.decodeValue()
		if err != nil {
			return nil, fmt.Errorf("Failed to decode value: %w", err)
		}
		vals = append(vals, val)
	}

	return vals, nil
}

func (d *Decoder) decodeValue() (*xmlrpc.Value, error) {
	var valueType uint32
	if err := binary.Read(d.b, binary.BigEndian, &valueType); err != nil {
		return nil, fmt.Errorf("Failed to decode value type: %w", err)
	}

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
	return nil, fmt.Errorf("Unkwon value type")
}

func (d *Decoder) decodeString() (*xmlrpc.Value, error) {
	var length uint32
	if err := binary.Read(d.b, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("Failed to decode value type: %w", err)
	}

	str := make([]byte, int(length))
	if err := binary.Read(d.b, binary.BigEndian, &str); err != nil {
		return nil, fmt.Errorf("Failed to decode string ")
	}

	return &xmlrpc.Value{
		FlatString: string(str),
	}, nil
}

func (d *Decoder) decodeInteger() (*xmlrpc.Value, error) {
	var val int32
	if err := binary.Read(d.b, binary.BigEndian, &val); err != nil {
		return nil, fmt.Errorf("Failed to decode value type: %w", err)
	}

	return &xmlrpc.Value{
		I4: strconv.Itoa(int(val)),
	}, nil
}

func (d *Decoder) decodeBool() (*xmlrpc.Value, error) {
	var val uint8
	if err := binary.Read(d.b, binary.BigEndian, &val); err != nil {
		return nil, fmt.Errorf("Failed to decode bool value: %w", err)
	}

	return &xmlrpc.Value{
		Boolean: strconv.Itoa(int(val)),
	}, nil
}

func (d *Decoder) decodeDouble() (*xmlrpc.Value, error) {
	var double struct {
		Man int32
		Exp int32
	}

	if err := binary.Read(d.b, binary.BigEndian, &double); err != nil {
		return nil, fmt.Errorf("Failed to decode double")
	}

	val := math.Pow(2, float64(double.Exp)) * float64(double.Man) / (1 << 30)
	val = math.Round(val*10000) / 10000

	return &xmlrpc.Value{
		Double: fmt.Sprintf("%g", val),
	}, nil
}

func (d *Decoder) decodeArray() (*xmlrpc.Value, error) {
	var length uint32
	if err := binary.Read(d.b, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("Failed to decode aray length: %w", err)
	}

	val := &xmlrpc.Value{
		Array: &xmlrpc.Array{
			Data: []*xmlrpc.Value{},
		},
	}

	vals, err := d.decodeParamValues(length)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode array values: %w", err)
	}

	val.Array.Data = vals

	return val, nil
}

func (d *Decoder) decodeStruct() (*xmlrpc.Value, error) {
	var length uint32
	if err := binary.Read(d.b, binary.BigEndian, &length); err != nil {
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
