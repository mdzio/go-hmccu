package binrpc

const (
	// BIN-RPC message types
	msgTypeRequest  = 0x00
	msgTypeResponse = 0x01
	msgTypeFault    = 0xFF

	// BIN-RPC data types
	integerType = 0x01
	booleanType = 0x02
	stringType  = 0x03
	doubleType  = 0x04

	arrayType  = 0x100
	structType = 0x101

	// Following types are currently not supported and not needed for CUxD
	// support.
	timeType   = 0x05
	binaryType = 0x06
)

var (
	binrpcMarker = [3]byte{'B', 'i', 'n'}
)

type header struct {
	Marker  [3]byte
	MsgType uint8
	MsgSize uint32
}
