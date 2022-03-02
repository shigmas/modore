package transport

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shigmas/modore/internal/apdu"
	"github.com/shigmas/modore/internal/npdu"
	"github.com/shigmas/modore/pkg/bacnet"
)

func TestBVLCCoding(t *testing.T) {

	expectedBytes := []byte{129, 11, 0, 13, 1, 0, 16, 8, 9, 0, 26, 3, 231}
	appMsg, err := apdu.NewWhoisMessage(0, 999)
	assert.NoError(t, err, "Unable to create WhoIs message")
	npduMsg := npdu.NewMessage(npdu.NormalMessage, false, false, nil, nil, DefaultHopCount, npdu.NetworkLayerWhoIsMessage, nil, appMsg)
	npduEncoded, err := npduMsg.Encode()
	assert.NoError(t, err, "Unable to create NPDU message")
	bvlcMsg := NewBVLCMessage(BVLCFunctioncBroadcast, npduEncoded)
	bvlcEncoded := bvlcMsg.Encode()
	assert.True(t, reflect.DeepEqual(expectedBytes, bvlcEncoded), "Encoding does not match expected")

	decodedMsg, err := NewBVLCMessageFromBytes(bvlcEncoded)
	assert.NoError(t, err, "Unable to decoded BVLC NPDU")
	assert.Equal(t, bvlcMsg.Function, decodedMsg.Function, "Decoded message function does not match")
	assert.True(t, reflect.DeepEqual(npduEncoded, decodedMsg.Data), "Decoded message contents do not match")
}

func TestBVLCDecoding(t *testing.T) {
	// Make sure we can handle more messages (that maybe we can't handle)
	testCases := []struct {
		name          string
		data          []byte
		expectedError error
	}{
		// These can be padded with 0's at the end
		{"EventNotImplemented", []byte{129, 11, 0, 19, 1, 32, 0, 0, 6, 186, 192, 255, 16, 8, 9, 0, 26, 3, 231, 0, 0, 0},
			bacnet.ErrNotImplemented},
		{"Thing", []byte{129, 11, 0, 20, 1, 32, 255, 255, 0, 255, 16, 8, 11, 63, 255, 255, 27, 63, 255, 255, 0, 0, 0, 0, 0},
			nil},
	}
	for _, tCase := range testCases {
		t.Run(tCase.name, func(t *testing.T) {
			decodedMsg, err := NewBVLCMessageFromBytes(tCase.data)
			assert.NoError(t, err, "Unexpected error in decoded")
			// BVLC should contain valid data (at least this does), so decode the contents too.
			npduMsg, err := npdu.NewMessageFromBytes(decodedMsg.Data)
			if tCase.expectedError != nil {
				assert.Equal(t, tCase.expectedError, err, "Error does not match")
			} else {
				fmt.Printf("type: %v\n", npduMsg.MessageType)
			}
		})
	}
}
