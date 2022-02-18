package bacnet

import (
	"bytes"
	"fmt"

	"github.com/shigmas/modore/pkg/apdu"
)

// BVLC is the BACnet Virtual Link Layer. It allows us to talk to any type of network. But, really,
// we only talk to IP.
// This encoding is in Section J of the spec.
//    7   6   5   4   3   2   1   0
//  |---|---|---|---|---|---|---|---|
//  | BVLC Type (only 1 type)       |
//  |---|---|---|---|---|---|---|---|
//  | Function                      |
//  |---|---|---|---|---|---|---|---|
//  | Length of data, including the |
//  | header. So, empty is 4 bytes) |
//  |---|---|---|---|---|---|---|---|
//  | Data                          |
//  |      .                        |
//  |      .                        |
//  |      .                        |
//  |---|---|---|---|---|---|---|---|

const (
	// BVLCType is the type of network. Only one type supported.
	BVLCType = 0x81
	// Header is 4 bytes
	BVLCHeaderLength = 4
)

// BVLCFunction is something that I need to research
type BVLCFunction uint8

// List of possible BACnet functions
const (
	BVLCFunctionResult                           BVLCFunction = 0
	BVLCFunctioncWriteBroadcastDistributionTable              = 1
	BVLCFunctioncBroadcastDistributionTable                   = 2
	BVLCFunctioncBroadcastDistributionTableAck                = 3
	BVLCFunctioncForwardedNPDU                                = 4
	BVLCFunctioncUnicast                                      = 10
	BVLCFunctioncBroadcast                                    = 11
)

type (
	// BVLCMessage has 4 pieces, only two of which are settable:
	// Type: There is // Length is also sent, but we will calculate it from the data.
	BVLCMessage struct {
		Function BVLCFunction
		Data     []byte
	}
)

// NewBVLCMessage creates a BVLCMessage
func NewBVLCMessage(function BVLCFunction, data []byte) *BVLCMessage {
	return &BVLCMessage{
		Function: function,
		Data:     data,
	}
}

func verifyFunction(maybe byte) bool {
	var val BVLCFunction = BVLCFunction(maybe)
	return val == BVLCFunctionResult ||
		val == BVLCFunctioncWriteBroadcastDistributionTable ||
		val == BVLCFunctioncBroadcastDistributionTable ||
		val == BVLCFunctioncBroadcastDistributionTableAck ||
		val == BVLCFunctioncForwardedNPDU ||
		val == BVLCFunctioncUnicast ||
		val == BVLCFunctioncBroadcast

}

func NewBVLCMessageFromBytes(encoded []byte) (*BVLCMessage, error) {
	buf := bytes.NewBuffer(encoded)
	var m BVLCMessage

	bvlcType, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	if bvlcType != BVLCType {
		return nil, fmt.Errorf("BVLCType mismatch. Received %d, expected %d", bvlcType, BVLCType)
	}
	b, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	if !verifyFunction(b) {
		return nil, fmt.Errorf("invalid value for BVLCFunction (%d)", m.Function)
	}
	lengthBytes := make([]byte, 2)
	bytesRead, err := buf.Read(lengthBytes)
	if err != nil {
		return nil, err
	}
	if bytesRead != 2 {
		return nil, fmt.Errorf("expected 2 bytes for length. Got %d", bytesRead)
	}
	msgLength := apdu.DecodeUint(lengthBytes)
	dataLength := msgLength - BVLCHeaderLength
	dataBytes := make([]byte, dataLength)
	bytesRead, err = buf.Read(dataBytes)
	if err != nil {
		return nil, err
	}
	if (uint)(bytesRead) != dataLength {
		return nil, fmt.Errorf("expected %d bytes for data. Got %d", bytesRead, dataLength)
	}
	return &BVLCMessage{
		Function: BVLCFunction(b),
		Data:     dataBytes,
	}, nil
}

// BVLCEncode encodes a BVLCMessage
func (m *BVLCMessage) Encode() []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 4))

	// all bytes, or uint8, except for length, so endian only matters for that.
	buf.WriteByte(BVLCType)
	buf.WriteByte((byte)(m.Function))
	// Possible loss of precision... but len is signed.
	// Include the length of the meta data
	buf.Write(apdu.EncodeUint((uint)(len(m.Data)+4), 2))
	buf.Write(m.Data)

	return buf.Bytes()
}

// BVLCDecode decodes a BVLCMessage
