package apdu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnsignedIntCoding(t *testing.T) {
	uTag, err := NewContextSpecificUnsignedInt(0, 381)
	assert.NoError(t, err, "Unexpected error")
	assert.NotNil(t, uTag, "Unable to create Context Specific Unsigned Int")

	uTag.EncodeAsTagData(TagContextSpecificClass)
	assert.NotNil(t, uTag, "Unable to encode int as Context Specific")

}
