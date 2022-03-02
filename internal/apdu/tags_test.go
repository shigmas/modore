package apdu

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shigmas/modore/pkg/bacnet"
)

func TestTagEncodings(t *testing.T) {
	t.Run("TestCodeTagNumber", func(t *testing.T) {
		testEncodeCases := []struct {
			name          string
			num           uint8
			expectedByte  byte
			expectedBytes []byte
			expectedError error
		}{
			{"fixed", 11, 0xB0, nil, nil},
			{"edge", 14, 0xE0, nil, nil},
			{"overflow", 20, 0xF0, []byte{20}, nil},
		}
		for _, tCase := range testEncodeCases {
			t.Run(tCase.name, func(t *testing.T) {
				var control byte

				trailingBytes := encodeTagNumber(&control, tCase.num)
				assert.Equal(t, tCase.expectedByte, control, "Control byte mismatch")
				assert.True(t, reflect.DeepEqual(tCase.expectedBytes, trailingBytes), "Trailing bytes mismatch")
			})
		}

		testDecodeCases := []struct {
			name          string
			expectedNum   uint8
			control       byte
			encodedBytes  []byte
			expectedError error
		}{
			{"fixed", 11, 0xB0, nil, nil},
			{"edge", 14, 0xE0, nil, nil},
			{"overflow", 20, 0xF0, []byte{20}, nil},
			{"overflowInsufficient", 0, 0xF0, nil, bacnet.ErrInsufficientData},
		}
		for _, tCase := range testDecodeCases {
			t.Run(tCase.name, func(t *testing.T) {

				tag, err := decodeTagNumber(tCase.control, bytes.NewBuffer(tCase.encodedBytes))
				assert.Equal(t, tCase.expectedNum, tag, "tag  mismatch")
				assert.Equal(t, tCase.expectedError, err, "Error mismatch")
			})
		}

	})
}

func TestClassEncoding(t *testing.T) {
	t.Run("TestEncodeClass", func(t *testing.T) {
		testEncodeCases := []struct {
			name         string
			class        TagClass
			existing     byte
			expectedByte byte
		}{
			{"application", TagApplicationClass, 0, 0},
			{"context specific", TagContextSpecificClass, 0, 8},
			{"context specific with tag", TagContextSpecificClass, 0xF0, 0xF8},
		}
		for _, tCase := range testEncodeCases {
			t.Run(tCase.name, func(t *testing.T) {
				encodeClass(&tCase.existing, tCase.class)
				assert.Equal(t, tCase.expectedByte, tCase.existing, "Unexpected value after encoding class")
			})
		}
	})
	t.Run("TestDecodeClass", func(t *testing.T) {
		testDecodeCases := []struct {
			name     string
			class    TagClass
			existing byte
		}{
			{"application", TagApplicationClass, 0xF1},
			{"context specific", TagContextSpecificClass, 8},
			{"context specific with tag", TagContextSpecificClass, 0xFF},
		}
		for _, tCase := range testDecodeCases {
			t.Run(tCase.name, func(t *testing.T) {
				res := decodeClass(tCase.existing)
				assert.Equal(t, res, tCase.class, "Unexpected value after decoding class")
			})
		}
	})
}

func TestLengthEncoding(t *testing.T) {
	t.Run("TestEncodeLength", func(t *testing.T) {
		testEncodeCases := []struct {
			name            string
			length          uint
			existingControl byte
			expectedControl byte
			expectedBytes   []byte
			expectedError   error
		}{
			{"inside byte application", 4, 0xF0, 0xF4, nil, nil},
			{"one byte", 201, 0, 0x05, []byte{201}, nil},
			{"one byte edge", 253, 0, 0x05, []byte{253}, nil},
			{"two byte small", 255, 0, 0x05, []byte{254, 0, 255}, nil},
			{"two byte large", 0xEFFF, 0, 0x05, []byte{254, 0xEF, 0xFF}, nil},
			{"four byte small", 0x1FFFF, 0, 0x05, []byte{255, 0, 0x01, 0xFF, 0xFF}, nil},
			{"four byte large", 0x1FABCDEF, 0, 0x05, []byte{255, 0x1F, 0xAB, 0xCD, 0xEF}, nil},
			{"value too large", 0xABCDEFABCD, 0, 0x05, nil, bacnet.ErrValueTooLarge},
		}
		for _, tCase := range testEncodeCases {
			t.Run(tCase.name, func(t *testing.T) {
				control := tCase.existingControl
				trailing, err := encodeLength(&control, tCase.length)
				if tCase.expectedError != nil {
					assert.Equal(t, tCase.expectedError, err)
				} else {
					assert.Equal(t, tCase.expectedControl, control, "Unexpected control byte after encoding length")
					assert.True(t, reflect.DeepEqual(tCase.expectedBytes, trailing),
						"Trailing bytes not equal")
				}

			})
		}

	})

	t.Run("TestDecodeLength", func(t *testing.T) {
		testEncodeCases := []struct {
			name           string
			control        byte
			data           []byte
			expectedLength uint
			expectedError  error
		}{
			{"inside byte", 0x04, nil, 4, nil},
			{"one byte error", 0x05, nil, 0, bacnet.ErrInsufficientData},
			{"one byte", 0x05, []byte{201}, 201, nil},
			{"one byte edge", 0x05, []byte{253}, 253, nil},
			{"two byte error", 0x05, []byte{254}, 0, bacnet.ErrInsufficientData},
			{"two byte small", 0x05, []byte{254, 0, 255}, 255, nil},
			{"two byte large", 0x05, []byte{254, 0xEF, 0xFF}, 0xEFFF, nil},
			{"two byte error", 0x05, []byte{255}, 0, bacnet.ErrInsufficientData},
			{"four byte small", 0x05, []byte{255, 0, 0x01, 0xFF, 0xFF}, 0x1FFFF, nil},
			{"four byte large", 0x05, []byte{255, 0x1F, 0xAB, 0xCD, 0xEF}, 0x1FABCDEF, nil},
		}
		for _, tCase := range testEncodeCases {
			t.Run(tCase.name, func(t *testing.T) {
				len, err := decodeLength(tCase.control, bytes.NewBuffer(tCase.data))
				if tCase.expectedError != nil {
					assert.Equal(t, tCase.expectedError, err)
				} else {
					assert.Equal(t, tCase.expectedLength, len, "Unexpected length")
				}
			})
		}

	})
}
