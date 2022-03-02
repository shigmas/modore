package apdu

import (
	"bytes"
	"reflect"
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

func TestBoolCoding(t *testing.T) {
	testCases := []struct {
		name          string
		tagNumber     uint8
		val           bool
		expectedBytes []byte
	}{
		{"True", 2, true, []byte{41, 1}},
		{"TrueOverflow", 96, true, []byte{249, 96, 1}},
		{"False", 1, false, []byte{25, 0}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tag, err := NewContextSpecificBool(tc.tagNumber, tc.val)
			assert.NoError(t, err, "Failed to create context specific bool")
			encodedTag, err := tag.EncodeAsTagData(TagContextSpecificClass)
			assert.NoError(t, err, "Failed to encode context specific bool")
			assert.True(t, reflect.DeepEqual(tc.expectedBytes, encodedTag), "Encoded bytes does not match expected")
			decodedTag, err := NewContextSpecificBoolFromBytes(bytes.NewBuffer(encodedTag))
			assert.NoError(t, err, "Failed to decode context specific bool")
			boolTag, ok := tag.(*ContextSpecificBoolType)
			assert.True(t, ok, "Could not cast to bool")
			decodedBool, ok := decodedTag.(*ContextSpecificBoolType)
			assert.Equal(t, boolTag.TagNumber, decodedBool.TagNumber, "original and decoded object tag do not match")
			assert.Equal(t, boolTag.val, decodedBool.val, "original and decoded value do not match")
		})
	}
}

func TestObjectIDCoding(t *testing.T) {
	testCases := []struct {
		name           string
		tagNumber      uint8
		objectType     uint32
		objectInstance uint32
		expectedBytes  []byte
	}{
		{"ObjectPassing", 2, 0x01, 0x32, []byte{44, 0, 64, 0, 50}},
		{"ObjectTypeTooBig", 2, 0x01, 0x32, nil},
		{"ObjectInstanceTooBig", 2, 0x01, 0x32, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tag, err := NewContextSpecificObjectID(tc.tagNumber, tc.objectType, tc.objectInstance)
			assert.NoError(t, err, "Failed to create context specific object id")
			encodedTag, err := tag.EncodeAsTagData(TagContextSpecificClass)
			assert.NoError(t, err, "Failed to encode context specific object id")
			//assert.True(t, reflect.DeepEqual(tc.expectedBytes, encodedTag), "Encoded bytes does not match expected")
			decodedTag, err := NewContextSpecificObjectIDFromBytes(bytes.NewBuffer(encodedTag))
			assert.NoError(t, err, "Failed to decode context specific object id")
			objectTag, ok := tag.(*ContextSpecificObjectIDType)
			assert.True(t, ok, "Could not cast to bool")
			decodedObject, ok := decodedTag.(*ContextSpecificObjectIDType)
			assert.Equal(t, objectTag.TagNumber, decodedObject.TagNumber, "original and decoded object tag do not match")
			assert.Equal(t, objectTag.objectType, decodedObject.objectType, "original and decoded object type do not match")
		})
	}
}
