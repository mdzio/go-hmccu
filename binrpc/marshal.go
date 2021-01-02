package binrpc

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/mdzio/go-hmccu/model"
	"io"
	"math"
	"strconv"
)

const (
	msgTypeRequest    = 0x00
	msgTypeResponse   = 0x01
	requestHeaderSize = 8
)

type Encoder struct {
	b        *bufio.Writer
	paramBuf *bytes.Buffer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		b:        bufio.NewWriter(w),
		paramBuf: &bytes.Buffer{},
	}
}

func (e *Encoder) EncodeRequest(method string, params []*model.Value) error {
	err := e.encodeParams(params)
	if err != nil {
		return err
	}

	contentSize := e.paramBuf.Len()

	_, err = e.b.Write([]byte("Bin"))
	if err != nil {
		return err
	}
	_, err = e.b.Write([]byte{msgTypeRequest})
	if err != nil {
		return fmt.Errorf("Failed to add msg type request: %w", err)
	}

	err = binary.Write(e.b, binary.BigEndian, int32(requestHeaderSize+len(method)+contentSize))
	if err != nil {
		return fmt.Errorf("Failed to add msg size: %w", err)
	}

	err = binary.Write(e.b, binary.BigEndian, uint32(len(method)))
	if err != nil {
		return fmt.Errorf("Failed to add method size: %w", err)
	}

	_, err = e.b.Write([]byte(method))
	if err != nil {
		return err
	}

	err = binary.Write(e.b, binary.BigEndian, uint32(len(params)))
	if err != nil {
		return fmt.Errorf("Failed to add params size: %w", err)
	}

	_, err = e.b.ReadFrom(e.paramBuf)
	if err != nil {
		return fmt.Errorf("Failed to add params: %w", err)
	}

	return e.b.Flush()
}

func (e *Encoder) EncodeResponse(param *model.Value) error {
	q := model.Q(param)
	if q.IsEmpty() {
		svrLog.Debugf("Encoding empty string response")
		err := e.encodeString("")
		if err != nil {
			return err
		}
	} else {
		err := e.encodeParam(param)
		if err != nil {
			return err
		}
	}

	contentSize := e.paramBuf.Len()

	_, err := e.b.Write([]byte("Bin"))
	if err != nil {
		return err
	}

	_, err = e.b.Write([]byte{msgTypeResponse})
	if err != nil {
		return fmt.Errorf("Failed to msg type response: %w", err)
	}
	err = binary.Write(e.b, binary.BigEndian, int32(contentSize))
	if err != nil {
		return fmt.Errorf("Failed to add msg size: %w", err)
	}

	_, err = e.b.ReadFrom(e.paramBuf)
	if err != nil {
		return fmt.Errorf("Failed to add param: %w", err)
	}

	return e.b.Flush()
}

func (e *Encoder) encodeParams(params []*model.Value) error {
	for _, v := range params {
		err := e.encodeParam(v)
		if err != nil {
			return fmt.Errorf("Failed to encode params: %w", err)
		}
	}
	return nil
}

func (e *Encoder) encodeParam(v *model.Value) error {
	switch {
	case v.String != "":
		err := e.encodeString(v.String)
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

func (e *Encoder) encodeStruct(v *model.Struct) error {
	err := binary.Write(e.paramBuf, binary.BigEndian, uint32(structType))
	if err != nil {
		return fmt.Errorf("Failed to add type struct: %w", err)
	}

	err = binary.Write(e.paramBuf, binary.BigEndian, uint32(len(v.Members)))
	if err != nil {
		return fmt.Errorf("Failed to add struct length: %w", err)
	}

	for _, m := range v.Members {
		err := e.encodeStringValue(m.Name)
		if err != nil {
			return fmt.Errorf("Failed to encode struct key: %w", err)
		}

		err = e.encodeParams([]*model.Value{m.Value})
		if err != nil {
			return fmt.Errorf("Failed to encode struct value: %w", err)
		}
	}

	return nil
}

func (e *Encoder) encodeString(str string) error {
	err := binary.Write(e.paramBuf, binary.BigEndian, uint32(stringType))
	if err != nil {
		return fmt.Errorf("Failed to add type string: %w", err)
	}

	err = e.encodeStringValue(str)
	if err != nil {
		return fmt.Errorf("Failed to add string value: %w", err)
	}

	return nil
}

func (e *Encoder) encodeStringValue(str string) error {
	err := binary.Write(e.paramBuf, binary.BigEndian, uint32(len(str)))
	if err != nil {
		return fmt.Errorf("Failed to add string size: %w", err)
	}

	_, err = e.paramBuf.Write([]byte(str))
	if err != nil {
		return err
	}
	return nil
}

func (e *Encoder) encodeInteger(n string) error {
	num, err := strconv.Atoi(n)
	if err != nil {
		return fmt.Errorf("Value is not an integer: %w", err)
	}

	err = binary.Write(e.paramBuf, binary.BigEndian, uint32(integerType))
	if err != nil {
		return fmt.Errorf("Failed to add type string: %w", err)
	}

	err = binary.Write(e.paramBuf, binary.BigEndian, int32(num))
	if err != nil {
		return err
	}

	return nil
}

func (e *Encoder) encodeDouble(v string) error {
	num, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("Value is not an int64: %w", err)
	}

	err = binary.Write(e.paramBuf, binary.BigEndian, uint32(doubleType))
	if err != nil {
		return fmt.Errorf("Failed to add type string: %w", err)
	}

	exp := math.Floor(math.Log(math.Abs(num))/math.Ln2) + 1
	man := math.Floor((num * math.Pow(2, -1*exp)) * (1 << 30))
	err = binary.Write(e.paramBuf, binary.BigEndian, int32(man))
	if err != nil {
		return err
	}

	err = binary.Write(e.paramBuf, binary.BigEndian, int32(exp))
	if err != nil {
		return err
	}

	return nil
}

func (e *Encoder) encodeBool(val string) error {
	var boolVal bool

	switch val {
	case "0":
		boolVal = false
	case "1":
		boolVal = true
	default:
		return fmt.Errorf("Value is not a bool")
	}

	err := binary.Write(e.paramBuf, binary.BigEndian, uint32(booleanType))
	if err != nil {
		return fmt.Errorf("Failed to add type bool: %w", err)
	}

	err = binary.Write(e.paramBuf, binary.BigEndian, boolVal)
	if err != nil {
		return err
	}

	return nil
}

func (e *Encoder) encodeArray(arr *model.Array) error {
	err := binary.Write(e.paramBuf, binary.BigEndian, uint32(arrayType))
	if err != nil {
		return fmt.Errorf("Failed to add type array: %w", err)
	}

	err = binary.Write(e.paramBuf, binary.BigEndian, uint32(len(arr.Data)))
	if err != nil {
		return fmt.Errorf("Failed to add array length: %w", err)
	}

	err = e.encodeParams(arr.Data)
	if err != nil {
		return fmt.Errorf("Failed to encode Array: %w", err)
	}

	return nil
}
