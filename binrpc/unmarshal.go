package binrpc

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/mdzio/go-hmccu/model"
	"io"
	"math"
	"strconv"
)

const (
	integerType = 0x01
	booleanType = 0x02
	stringType  = 0x03
	doubleType  = 0x04
	arrayType   = 0x100
	structType  = 0x101
)

type Decoder struct {
	b *bufio.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		b: bufio.NewReader(r),
	}
}

func (d *Decoder) DecodeRequest() (string, []*model.Value, error) {
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

func (d *Decoder) DecodeResponse() (*model.Value, error) {
	var header struct {
		Head    [3]byte
		MsgType uint8
		MsgSize uint32
	}

	if err := binary.Read(d.b, binary.BigEndian, &header); err != nil {
		return nil, fmt.Errorf("Failed to decode header")
	}

	return d.decodeValue()
}

func (d *Decoder) decodeParams() ([]*model.Value, error) {
	var elementCount uint32
	if err := binary.Read(d.b, binary.BigEndian, &elementCount); err != nil {
		return nil, fmt.Errorf("Failed to decode element count ")
	}

	return d.decodeParamValues(elementCount)

}

func (d *Decoder) decodeParamValues(elementCount uint32) ([]*model.Value, error) {
	vals := []*model.Value{}
	for i := 0; i < int(elementCount); i++ {
		val, err := d.decodeValue()
		if err != nil {
			return nil, fmt.Errorf("Failed to decode value: %w", err)
		}
		vals = append(vals, val)
	}

	return vals, nil
}

func (d *Decoder) decodeValue() (*model.Value, error) {
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

func (d *Decoder) decodeString() (*model.Value, error) {
	var length uint32
	if err := binary.Read(d.b, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("Failed to decode value type: %w", err)
	}

	str := make([]byte, int(length))
	if err := binary.Read(d.b, binary.BigEndian, &str); err != nil {
		return nil, fmt.Errorf("Failed to decode string ")
	}

	return &model.Value{
		FlatString: string(str),
	}, nil
}

func (d *Decoder) decodeInteger() (*model.Value, error) {
	var val int32
	if err := binary.Read(d.b, binary.BigEndian, &val); err != nil {
		return nil, fmt.Errorf("Failed to decode value type: %w", err)
	}

	return &model.Value{
		I4: strconv.Itoa(int(val)),
	}, nil
}

func (d *Decoder) decodeBool() (*model.Value, error) {
	var val uint8
	if err := binary.Read(d.b, binary.BigEndian, &val); err != nil {
		return nil, fmt.Errorf("Failed to decode bool value: %w", err)
	}

	return &model.Value{
		Boolean: strconv.Itoa(int(val)),
	}, nil
}

func (d *Decoder) decodeDouble() (*model.Value, error) {
	var double struct {
		Man int32
		Exp int32
	}

	if err := binary.Read(d.b, binary.BigEndian, &double); err != nil {
		return nil, fmt.Errorf("Failed to decode double")
	}

	val := math.Pow(2, float64(double.Exp)) * float64(double.Man) / (1 << 30)
	val = math.Round(val*10000) / 10000

	return &model.Value{
		Double: fmt.Sprintf("%g", val),
	}, nil
}

func (d *Decoder) decodeArray() (*model.Value, error) {
	var length uint32
	if err := binary.Read(d.b, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("Failed to decode aray length: %w", err)
	}

	val := &model.Value{
		Array: &model.Array{
			Data: []*model.Value{},
		},
	}

	vals, err := d.decodeParamValues(length)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode array values: %w", err)
	}

	val.Array.Data = vals

	return val, nil
}

func (d *Decoder) decodeStruct() (*model.Value, error) {
	var length uint32
	if err := binary.Read(d.b, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("Failed to decode struct length: %w", err)
	}

	val := &model.Value{
		Struct: &model.Struct{Members: []*model.Member{}},
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
		val.Struct.Members = append(val.Struct.Members, &model.Member{
			Name:  keyVal.FlatString,
			Value: structVal,
		})
	}

	return val, nil
}
