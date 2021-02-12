package binrpc

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"golang.org/x/text/encoding/charmap"
)

// Encoder encodes XML-RPC requests as BIN-RPC.
type Encoder struct {
	w *bufio.Writer
}

// NewEncoder creates an encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: bufio.NewWriter(w)}
}

// EncodeRequest encodes a XML-RPC request.
func (e *Encoder) EncodeRequest(method string, params []*xmlrpc.Value) error {
	// encode parameters
	pe := valueEncoder{}
	err := pe.encodeParams(params)
	if err != nil {
		return err
	}

	// encode method name
	me := valueEncoder{}
	err = me.encodeStringWOType(method)
	if err != nil {
		return err
	}

	// calculate payload size
	payloadSize := me.Len() /* method name */ + pe.Len() /* params */

	// write header
	_, err = e.w.Write(binrpcMarker[:])
	if err != nil {
		return err
	}
	_, err = e.w.Write([]byte{msgTypeRequest})
	if err != nil {
		return fmt.Errorf("Writing of message type failed: %w", err)
	}
	err = binary.Write(e.w, binary.BigEndian, int32(payloadSize))
	if err != nil {
		return fmt.Errorf("Writing of payload size failed: %w", err)
	}

	// write method name and parameters
	_, err = e.w.ReadFrom(io.MultiReader(&me, &pe))
	if err != nil {
		return fmt.Errorf("Writing of method name or parameters failed: %w", err)
	}
	return e.w.Flush()
}

// EncodeResponse encodes a XML-RPC response.
func (e *Encoder) EncodeResponse(value *xmlrpc.Value) error {
	// encode value
	ve := valueEncoder{}
	q := xmlrpc.Q(value)
	if q.IsEmpty() {
		err := ve.encodeString("")
		if err != nil {
			return err
		}
	} else {
		err := ve.encodeValue(value)
		if err != nil {
			return err
		}
	}

	// write header
	_, err := e.w.Write(binrpcMarker[:])
	if err != nil {
		return err
	}
	_, err = e.w.Write([]byte{msgTypeResponse})
	if err != nil {
		return fmt.Errorf("Writing of message type failed: %w", err)
	}
	err = binary.Write(e.w, binary.BigEndian, int32(ve.Len()))
	if err != nil {
		return fmt.Errorf("Writing of payload size failed: %w", err)
	}

	// write value
	_, err = e.w.ReadFrom(&ve)
	if err != nil {
		return fmt.Errorf("Writing of value failed: %w", err)
	}
	return e.w.Flush()
}

type valueEncoder struct {
	bytes.Buffer
}

func (e *valueEncoder) encodeParams(params []*xmlrpc.Value) error {
	// write number of parameters
	err := binary.Write(e, binary.BigEndian, uint32(len(params)))
	if err != nil {
		return fmt.Errorf("Writing number of parameters failed: %w", err)
	}

	// write parameters
	for _, v := range params {
		err := e.encodeValue(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *valueEncoder) encodeValue(v *xmlrpc.Value) error {
	switch {
	case v.ElemString != "":
		err := e.encodeString(v.ElemString)
		if err != nil {
			return fmt.Errorf("Failed to encode string: %w", err)
		}
	case v.FlatString != "":
		err := e.encodeString(v.FlatString)
		if err != nil {
			return fmt.Errorf("Failed to encode flatstring: %w", err)
		}
	case v.Int != "":
		err := e.encodeInteger(v.Int)
		if err != nil {
			return fmt.Errorf("Failed to encode integer: %w", err)
		}
	case v.I4 != "":
		err := e.encodeInteger(v.I4)
		if err != nil {
			return fmt.Errorf("Failed to encode i4: %w", err)
		}
	case v.Boolean != "":
		err := e.encodeBool(v.Boolean)
		if err != nil {
			return fmt.Errorf("Failed to encode bool: %w", err)
		}
	case v.Double != "":
		err := e.encodeDouble(v.Double)
		if err != nil {
			return fmt.Errorf("Failed to encode double: %w", err)
		}
	case v.Struct != nil:
		err := e.encodeStruct(v.Struct)
		if err != nil {
			return fmt.Errorf("Failed to encode struct: %w", err)
		}
	case v.Array != nil:
		err := e.encodeArray(v.Array)
		if err != nil {
			return fmt.Errorf("Failed to encode array: %w", err)
		}
	default:
		err := e.encodeString("")
		if err != nil {
			return fmt.Errorf("Failed to encode empty value as flatstring: %w", err)
		}

	}

	return nil
}

func (e *valueEncoder) encodeStruct(v *xmlrpc.Struct) error {
	err := binary.Write(e, binary.BigEndian, uint32(structType))
	if err != nil {
		return fmt.Errorf("Failed to add type struct: %w", err)
	}

	err = binary.Write(e, binary.BigEndian, uint32(len(v.Members)))
	if err != nil {
		return fmt.Errorf("Failed to add struct length: %w", err)
	}

	for _, m := range v.Members {
		err := e.encodeStringWOType(m.Name)
		if err != nil {
			return fmt.Errorf("Failed to encode struct key: %w", err)
		}

		err = e.encodeValue(m.Value)
		if err != nil {
			return fmt.Errorf("Failed to encode struct value: %w", err)
		}
	}

	return nil
}

func (e *valueEncoder) encodeString(str string) error {
	// write data type
	err := binary.Write(e, binary.BigEndian, uint32(stringType))
	if err != nil {
		return fmt.Errorf("Writing of string type failed: %w", err)
	}

	// write length and content
	err = e.encodeStringWOType(str)
	if err != nil {
		return err
	}
	return nil
}

func (e *valueEncoder) encodeStringWOType(str string) error {
	// encode string with ISO8859-1
	buf := bytes.Buffer{}
	charmap.ISO8859_1.NewEncoder().Writer(&buf).Write([]byte(str))

	// write length
	err := binary.Write(e, binary.BigEndian, uint32(len(buf.Bytes())))
	if err != nil {
		return fmt.Errorf("Writing of string length failed: %w", err)
	}

	// write content
	_, err = e.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("Writing of string content failed: %w", err)
	}
	return nil
}

func (e *valueEncoder) encodeInteger(n string) error {
	// convert string to integer
	num, err := strconv.Atoi(n)
	if err != nil {
		return fmt.Errorf("Invalid integer value: %s", n)
	}

	// write data type
	err = binary.Write(e, binary.BigEndian, uint32(integerType))
	if err != nil {
		return fmt.Errorf("Writing of integer type failed: %w", err)
	}

	// write integer
	err = binary.Write(e, binary.BigEndian, int32(num))
	if err != nil {
		return fmt.Errorf("Writing of integer failed: %w", err)
	}
	return nil
}

func (e *valueEncoder) encodeDouble(v string) error {
	// convert string to float64
	num, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("Invalid float value: %s", v)
	}

	// write data type
	err = binary.Write(e, binary.BigEndian, uint32(doubleType))
	if err != nil {
		return fmt.Errorf("Writing of double type failed: %w", err)
	}

	// convert to BIN-RPC representation
	exp := math.Floor(math.Log(math.Abs(num))/math.Ln2) + 1
	man := math.Floor((num * math.Pow(2, -1*exp)) * (1 << 30))

	// write BIN-RPC representation
	err = binary.Write(e, binary.BigEndian, int32(man))
	if err != nil {
		return fmt.Errorf("Writing of double mantissa failed: %w", err)
	}
	err = binary.Write(e, binary.BigEndian, int32(exp))
	if err != nil {
		return fmt.Errorf("Writing of double exponent failed: %w", err)
	}
	return nil
}

func (e *valueEncoder) encodeBool(val string) error {
	// convert string to bool
	var boolVal bool
	switch val {
	case "0":
		boolVal = false
	case "1":
		boolVal = true
	default:
		return fmt.Errorf("Invalid bool value: %s", val)
	}

	// write data type
	err := binary.Write(e, binary.BigEndian, uint32(booleanType))
	if err != nil {
		return fmt.Errorf("Writing of bool type failed: %w", err)
	}

	// write bool
	err = binary.Write(e, binary.BigEndian, boolVal)
	if err != nil {
		return fmt.Errorf("Writing of bool failed: %w", err)
	}
	return nil
}

func (e *valueEncoder) encodeArray(arr *xmlrpc.Array) error {
	// write data type
	err := binary.Write(e, binary.BigEndian, uint32(arrayType))
	if err != nil {
		return fmt.Errorf("Writing of array type failed: %w", err)
	}

	// encode array elements
	err = e.encodeParams(arr.Data)
	if err != nil {
		return err
	}
	return nil
}
