package bacnet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUInt(t *testing.T) {
	testCases := []struct {
		name     string
		val      uint
		wantSize uint
	}{
		{"nibble", 0x03, 1},
		{"byte", 0x11, 1},
		{"3nibbles", 0xABC, 2},
		{"2bytes", 0x00BADF, 2},
		{"3bytes", 0x00FA38BC, 3},
		{"4bytes", 0xFA38B89C, 4},
		{"5bytes", 0xFA3800BC99, 5},
		{"6bytes", 0x00FA38BCDD4477, 6},
		{"7bytes", 0x0000000FA38BCAAEEFF88, 7},
		{"8bytes", 0xFA38BCAABBCCDDEE, 8},
	}

	for _, tcase := range testCases {
		t.Run(tcase.name, func(t *testing.T) {
			assert.Equal(t, tcase.wantSize, GetUnsignedIntByteSize(tcase.val))

			b := EncodeUint(tcase.val, tcase.wantSize)
			assert.Equal(t, int(tcase.wantSize), len(b))
			decoded := DecodeUint(b)
			assert.Equal(t, tcase.val, decoded)
		})
	}
}
