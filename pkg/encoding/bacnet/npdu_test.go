package bacnet

import (
	"bytes"

	"testing"

	"github.com/stretchr/testify/assert"
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

func TestNPDUControl(t *testing.T) {
	t.Run("TestCreateNPDUControl", func(t *testing.T) {
		ctrl := NewNPDUControl(UrgentMessage, false, true, false, true)
		assert.False(t, ctrl.IsBACNetConfirmedRequestPDUPresent, "Unexpected value")
		assert.True(t, ctrl.SourceSpecifiersPresent, "Unexpected value")
		assert.False(t, ctrl.DestinationSpecifiersPresent, "Unexpected value")
		assert.True(t, ctrl.IsNDSUNetworkLayerMessage, "Unexpected value")

		t.Run("TestEncodeDecodeControl", func(t *testing.T) {
			encoded := encodeNPDUControl(*ctrl)
			// Verify that the two reserve bits are 0
			assert.Equal(t, uint8(0b10001001), encoded, "Unexpected encoding")

			decodedCtrl := decodeNPDUControl(encoded)
			assert.Equal(t, ctrl.IsBACNetConfirmedRequestPDUPresent, decodedCtrl.IsBACNetConfirmedRequestPDUPresent,
				"unexpected value")
			assert.Equal(t, ctrl.SourceSpecifiersPresent, decodedCtrl.SourceSpecifiersPresent, "Unexpected value")
			assert.Equal(t, ctrl.DestinationSpecifiersPresent, decodedCtrl.DestinationSpecifiersPresent, "Unexpected value")
			assert.Equal(t, ctrl.IsNDSUNetworkLayerMessage, decodedCtrl.IsNDSUNetworkLayerMessage, "Unexpected value")
			assert.Equal(t, ctrl.Priority, decodedCtrl.Priority, "Unexpected value")
		})

	})
}

func compareSlice(a, b []byte) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	for i, e := range a {
		if b[i] != e {
			return false
		}
	}
	return true
}

func TestSpecifics(t *testing.T) {
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
			spec := Specifics{
				Network: tCase.network,
				Length:  tCase.length,
				Address: tCase.address,
			}
			wBuf := bytes.NewBuffer(make([]byte, 0, 2))
			assert.NoError(t, writeSpecifics(wBuf, &spec), "Unexpected error")
			rBuf := bytes.NewBuffer(wBuf.Bytes())
			readSpec, err := readSpecifics(rBuf)
			assert.NoError(t, err, "Unexpected error")
			assert.Equal(t, spec.Network, readSpec.Network, "Value mismatch")
			assert.Equal(t, spec.Length, readSpec.Length, "Value mismatch")
			assert.True(t, compareSlice(spec.Address, readSpec.Address), "Value mismatch")
		})
	}
}
