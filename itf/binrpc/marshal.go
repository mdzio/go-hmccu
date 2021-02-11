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
	ve := valueEncoder{}
	err := ve.encodeParams(params)
	if err != nil {
		return err
	}
	payloadSize := 4 /* method len */ + len(method) + 4 /* params len */ + ve.Len()

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

	// write method name
	// TODO: fix encoding
	err = binary.Write(e.w, binary.BigEndian, uint32(len(method)))
	if err != nil {
		return fmt.Errorf("Writing of method length failed: %w", err)
	}
	_, err = e.w.Write([]byte(method))
	if err != nil {
		return fmt.Errorf("Writing of method name failed: %w", err)
	}

	// write number of parameters
	err = binary.Write(e.w, binary.BigEndian, uint32(len(params)))
	if err != nil {
		return fmt.Errorf("Writing number of parameters failed: %w", err)
	}

	// write parameters
	_, err = e.w.ReadFrom(&ve)
	if err != nil {
		return fmt.Errorf("Writing of parameters failed: %w", err)
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
	for _, v := range params {
		err := e.encodeValue(v)
		if err != nil {
			return fmt.Errorf("Failed to encode params: %w", err)
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

		err = e.encodeParams([]*xmlrpc.Value{m.Value})
		if err != nil {
			return fmt.Errorf("Failed to encode struct value: %w", err)
		}
	}

	return nil
}

func (e *valueEncoder) encodeString(str string) error {
	err := binary.Write(e, binary.BigEndian, uint32(stringType))
	if err != nil {
		return fmt.Errorf("Failed to add type string: %w", err)
	}

	err = e.encodeStringWOType(str)
	if err != nil {
		return fmt.Errorf("Failed to add string value: %w", err)
	}

	return nil
}

func (e *valueEncoder) encodeStringWOType(str string) error {
	err := binary.Write(e, binary.BigEndian, uint32(len(str)))
	if err != nil {
		return fmt.Errorf("Failed to add string size: %w", err)
	}

	// TODO: fix encoding
	_, err = e.Write([]byte(str))
	if err != nil {
		return err
	}
	return nil
}

func (e *valueEncoder) encodeInteger(n string) error {
	num, err := strconv.Atoi(n)
	if err != nil {
		return fmt.Errorf("Value is not an integer: %w", err)
	}

	err = binary.Write(e, binary.BigEndian, uint32(integerType))
	if err != nil {
		return fmt.Errorf("Failed to add type string: %w", err)
	}

	err = binary.Write(e, binary.BigEndian, int32(num))
	if err != nil {
		return err
	}

	return nil
}

func (e *valueEncoder) encodeDouble(v string) error {
	num, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("Value is not an int64: %w", err)
	}

	err = binary.Write(e, binary.BigEndian, uint32(doubleType))
	if err != nil {
		return fmt.Errorf("Failed to add type string: %w", err)
	}

	exp := math.Floor(math.Log(math.Abs(num))/math.Ln2) + 1
	man := math.Floor((num * math.Pow(2, -1*exp)) * (1 << 30))
	err = binary.Write(e, binary.BigEndian, int32(man))
	if err != nil {
		return err
	}

	err = binary.Write(e, binary.BigEndian, int32(exp))
	if err != nil {
		return err
	}

	return nil
}

func (e *valueEncoder) encodeBool(val string) error {
	var boolVal bool

	switch val {
	case "0":
		boolVal = false
	case "1":
		boolVal = true
	default:
		return fmt.Errorf("Value is not a bool")
	}

	err := binary.Write(e, binary.BigEndian, uint32(booleanType))
	if err != nil {
		return fmt.Errorf("Failed to add type bool: %w", err)
	}

	err = binary.Write(e, binary.BigEndian, boolVal)
	if err != nil {
		return err
	}

	return nil
}

func (e *valueEncoder) encodeArray(arr *xmlrpc.Array) error {
	err := binary.Write(e, binary.BigEndian, uint32(arrayType))
	if err != nil {
		return fmt.Errorf("Failed to add type array: %w", err)
	}

	err = binary.Write(e, binary.BigEndian, uint32(len(arr.Data)))
	if err != nil {
		return fmt.Errorf("Failed to add array length: %w", err)
	}

	err = e.encodeParams(arr.Data)
	if err != nil {
		return fmt.Errorf("Failed to encode Array: %w", err)
	}

	return nil
}
