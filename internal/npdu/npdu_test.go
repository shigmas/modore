package npdu

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shigmas/modore/internal/apdu"
)

func TestDoubleByte(t *testing.T) {
	t.Run("TestReadWriteDoubleByte", func(t *testing.T) {
		testCases := []struct {
			name string
			val  uint16
		}{
			{"small", 0x3F},
			{"smallmax", 0xFF},
			{"largemin", 0xF3D4},
			{"largemax", 0xFFFF},
		}
		for _, oneT := range testCases {
			t.Run(oneT.name, func(t *testing.T) {
				b := make([]byte, 0, 2)
				outBuf := bytes.NewBuffer(b)
				err := writeDoubleByte(outBuf, oneT.val)
				assert.NoError(t, err, "Unexpected error writing")
				inBuf := bytes.NewBuffer(outBuf.Bytes())
				inVal, err := readDoubleByte(inBuf)
				assert.NoError(t, err, "Unexpected error reading")
				assert.Equal(t, inVal, oneT.val, "Values did not match")
			})
		}
	})
}

func TestNPDUCoding(t *testing.T) {
	// Should flesh this out with more test cases.
	data := []byte{1, 0, 16, 8, 9, 0, 26, 3, 231}
	// Create an APDU message to send.
	appMsg, err := apdu.NewWhoisMessage(0, 999)
	assert.NoError(t, err, "Unexpected error creating test APDU Message")

	npduMsg := NewMessage(NormalMessage, false, false, nil, nil, 0xFF, NetworkLayerWhoIsMessage, nil, appMsg)

	npduBytes, err := npduMsg.Encode()
	assert.NoError(t, err, "Unexpected error encoding NPDU Message")
	assert.Equal(t, data, npduBytes, "Encoding not expected")

	npduDecoded, err := NewMessageFromBytes(npduBytes)
	assert.NoError(t, err, "Unable to decode valid message")
	// There are some differences, since we we conditionally encode/decode some fields (namely, hop count)
	assert.Equal(t, npduMsg.ProtocolVersion, npduDecoded.ProtocolVersion, "ProtocolVersion did not match")
	assert.Equal(t, npduMsg.Destination, npduDecoded.Destination, "Destination did not match")
	assert.Equal(t, npduMsg.Source, npduDecoded.Source, "Source did not match")
	assert.Equal(t, npduMsg.MessageType, npduDecoded.MessageType, "MessageType did not match")
	assert.Equal(t, npduMsg.VendorID, npduDecoded.VendorID, "VendorID did not match")
	assert.Equal(t, npduMsg.APDU, npduDecoded.APDU, "APDU did not match")

}

func TestControl(t *testing.T) {
	t.Run("TestCreateControl", func(t *testing.T) {
		ctrl := newControl(UrgentMessage, false, true, false, true)
		assert.False(t, ctrl.IsBACNetConfirmedRequestPDUPresent, "Unexpected value")
		assert.True(t, ctrl.SourceAddressPresent, "Unexpected value")
		assert.False(t, ctrl.DestinationAddressPresent, "Unexpected value")
		assert.True(t, ctrl.IsNDSUNetworkLayerMessage, "Unexpected value")

		t.Run("TestEncodeDecodeControl", func(t *testing.T) {
			encoded := encodeControl(ctrl)
			// Verify that the two reserve bits are 0
			assert.Equal(t, uint8(0b10001001), encoded, "Unexpected encoding")

			decodedCtrl := decodeControl(encoded)
			assert.Equal(t, ctrl.IsBACNetConfirmedRequestPDUPresent, decodedCtrl.IsBACNetConfirmedRequestPDUPresent,
				"unexpected value")
			assert.Equal(t, ctrl.SourceAddressPresent, decodedCtrl.SourceAddressPresent, "Unexpected value")
			assert.Equal(t, ctrl.DestinationAddressPresent, decodedCtrl.DestinationAddressPresent, "Unexpected value")
			assert.Equal(t, ctrl.IsNDSUNetworkLayerMessage, decodedCtrl.IsNDSUNetworkLayerMessage, "Unexpected value")
			assert.Equal(t, ctrl.Priority, decodedCtrl.Priority, "Unexpected value")
		})

	})
}

func TestAddress(t *testing.T) {
	testCases := []struct {
		name    string
		network uint16
		length  uint8
		address []byte
	}{
		{"TestEmptyAddress", 34, 0, nil},
		{"TestValidAddress", 255, 1, []byte{3}},
		{"TestAnotherValidAddress", 265, 8, []byte{8, 7, 6, 5, 4, 3, 2, 1}},
	}
	for _, tCase := range testCases {
		t.Run(tCase.name, func(t *testing.T) {
			spec := Address{
				Network: tCase.network,
				Length:  tCase.length,
				Addr:    tCase.address,
			}
			wBuf := bytes.NewBuffer(make([]byte, 0, 2))
			assert.NoError(t, writeAddress(wBuf, &spec), "Unexpected error")
			rBuf := bytes.NewBuffer(wBuf.Bytes())
			readSpec, err := readAddress(rBuf)
			assert.NoError(t, err, "Unexpected error")
			assert.Equal(t, spec.Network, readSpec.Network, "Value mismatch")
			assert.Equal(t, spec.Length, readSpec.Length, "Value mismatch")
			assert.True(t, reflect.DeepEqual(spec.Addr, readSpec.Addr), "Value mismatch")
		})
	}
}
