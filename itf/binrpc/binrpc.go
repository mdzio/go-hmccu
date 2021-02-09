package binrpc

const (
	requestHeaderSize = 8

	msgTypeRequest  = 0x00
	msgTypeResponse = 0x01
	msgTypeFault    = 0xFF

	integerType = 0x01
	booleanType = 0x02
	stringType  = 0x03
	doubleType  = 0x04

	// Following types are currently not supported and not needed for CUxD
	// support.
	timeType   = 0x05
	binaryType = 0x06

	arrayType  = 0x100
	structType = 0x101
)

var (
	binrpcMarker = [3]byte{'B', 'i', 'n'}
)

type header struct {
	Marker  [3]byte
	MsgType uint8
	MsgSize uint32
}
