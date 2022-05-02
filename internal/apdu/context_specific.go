package apdu

import (
	"bytes"
	"fmt"

	"github.com/shigmas/modore/pkg/bacnet"
)

// NewContextSpecificTagFromUint32 creates a tag for an uint32
// Note: even though this takes a uint32, the spec says that it should be in the smallest number of bytes
// possible.
// func NewContextSpecificTagFromUint(d uint, tagNumber uint8) *ContextSpecificTag {

// 	return &ContextSpecificTag{
// 		ParameterBase: ParameterBase{
// 			Type:     TagContextSpecificClass,
// 			DataType: TagNumberDataUnsignedInt,
// 		},
// 		DataLength: GetUnsignedIntByteSize(d),
// 		Data:       d,
// 	}
// }

// The control byte is:
//    7   6   5   4   3   2   1   0
//  |---|---|---|---|---|---|---|---|
//  | Tag           |cls| "length"  |
// Where tag is a number or a type, but called a type, class is a bit, and length is length, or maybe
// data. But, we let the caller decide. It

// context specific encoding (as opposed to application)
type (
	ContextSpecificTypeBase struct {
		TagTypeBase
		// TagNumber is a kind of ordering of the parameters in an APDU message
		TagNumber uint8
	}

	ContextSpecificNullType struct {
		ContextSpecificTypeBase
	}

	// ContextSpecificBoolType has content length of one, and the next byte is 0 (false) or 1 (true)
	ContextSpecificBoolType struct {
		ContextSpecificTypeBase
		val bool
	}

	// ContextSpecificUnsignedIntType is quite complicated, with the control byte being a flag or length.
	ContextSpecificUnsignedIntType struct {
		ContextSpecificTypeBase
		val uint
	}

	ContextSpecificSignedIntType struct {
	}
	ContextSpecificRealType struct {
	}
	ContextSpecificDoubleType struct {
	}
	ContextSpecificOctetStringType struct {
	}
	ContextSpecificCharacterStringType struct {
	}
	ContextSpecificBitStringType struct {
	}
	ContextSpecificEnumeratedType struct {
	}
	ContextSpecificDateType struct {
	}
	ContextSpecificTimeType struct {
	}

	// ContextSpecificObjectIDType is 2 values which total 32 bits.
	// 10 bits for the object type
	// 22 bits for the object instance number
	// Both go from the most significant to least. e.g. 9th or 21st most significant to 0 least significant
	//    31 ......22  21 ........... 0
	//  |------------|----------------|
	//  | Object Type| Object Instance|
	ContextSpecificObjectIDType struct {
		ContextSpecificTypeBase
		// In encoding, we will and these numbers, so just keep them as 32 bit, even though type will fit in
		// 16
		objectType     uint32
		objectInstance uint32
	}
)

func newContextSpecificTypeBase(tagNumber uint8) ContextSpecificTypeBase {
	return ContextSpecificTypeBase{TagNumber: tagNumber}
}

// NewContextSpecificUnsignedInt creates an unsigned int, for the tag
func NewContextSpecificUnsignedInt(tagNumber uint8, val uint) (TagType, error) {
	return &ContextSpecificUnsignedIntType{
		ContextSpecificTypeBase: newContextSpecificTypeBase(tagNumber),
		val:                     val,
	}, nil

}

// NewContextSpecificUnsignedIntFromBytes decodes the bytes into an context specific unsigned int
func NewContextSpecificUnsignedIntFromBytes(tagBuf *bytes.Buffer) (TagType, error) {
	control, err := tagBuf.ReadByte()
	if err != nil {
		return nil, err
	}
	tagNumber, err := decodeTagNumber(control, tagBuf)
	if err != nil {
		return nil, err
	}
	tagClass := decodeClass(control)
	if tagClass != TagContextSpecificClass {
		return nil, fmt.Errorf("Expected ContextSpecificTagClass")
	}
	tagLen, err := decodeLength(control, tagBuf)
	if err != nil {
		return nil, err
	}
	valBuf := make([]byte, tagLen)
	bytesRead, err := tagBuf.Read(valBuf)
	if err != nil {
		return nil, err
	} else if uint(bytesRead) != tagLen {
		return nil, bacnet.ErrInsufficientData
	}

	return &ContextSpecificUnsignedIntType{
		ContextSpecificTypeBase: newContextSpecificTypeBase(tagNumber),
		val:                     DecodeUint(valBuf),
	}, nil

}

// EncodeAsTagData encodes this unsigned int into the bytes
func (p *ContextSpecificUnsignedIntType) EncodeAsTagData(class TagClass) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 1))

	var control byte
	tagBytes := encodeTagNumber(&control, (uint8)(p.TagNumber))
	encodeClass(&control, TagContextSpecificClass)

	length := getUnsignedIntByteSize(p.val)
	lengthBytes, err := encodeLength(&control, length)
	if err != nil {
		return nil, err
	}
	encodedVal := EncodeUint(p.val, length)

	buf.WriteByte(control)
	if tagBytes != nil {
		// This is overflow tag number, if it exceeds what fits in the control byte
		buf.Write(tagBytes)
	}
	if lengthBytes != nil {
		// the number bytes did not fit into the length section of the control byte, so control byte
		// has a flag set and this has the length.
		buf.Write(lengthBytes)
	}

	buf.Write(encodedVal)
	return buf.Bytes(), nil
}

// NewContextSpecificBool creates the bool type with the tag
func NewContextSpecificBool(tagNumber uint8, val bool) (TagType, error) {
	return &ContextSpecificBoolType{
		ContextSpecificTypeBase: newContextSpecificTypeBase(tagNumber),
		val:                     val,
	}, nil

}

// NewContextSpecificBoolFromBytes decodes the bool type from bytes
func NewContextSpecificBoolFromBytes(tagBuf *bytes.Buffer) (TagType, error) {
	control, err := tagBuf.ReadByte()
	if err != nil {
		return nil, err
	}
	tagNumber, err := decodeTagNumber(control, tagBuf)
	if err != nil {
		return nil, err
	}
	tagClass := decodeClass(control)
	if tagClass != TagContextSpecificClass {
		return nil, fmt.Errorf("Expected ContextSpecificTagClass")
	}
	tagLen, err := decodeLength(control, tagBuf)
	if err != nil {
		return nil, err
	}
	if tagLen != 1 {
		return nil, bacnet.ErrInvalidData
	}

	bytesRead, err := tagBuf.ReadByte()
	if err != nil {
		return nil, err
	}
	val := false
	if bytesRead == 1 {
		val = true
	}
	return &ContextSpecificBoolType{
		ContextSpecificTypeBase: newContextSpecificTypeBase(tagNumber),
		val:                     val,
	}, nil
}

// EncodeAsTagData encodes the bool as bytes
func (p *ContextSpecificBoolType) EncodeAsTagData(class TagClass) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 1))

	var control byte
	tagBytes := encodeTagNumber(&control, (uint8)(p.TagNumber))
	encodeClass(&control, TagContextSpecificClass)

	lengthBytes, err := encodeLength(&control, 1)
	if err != nil {
		return nil, err
	}

	buf.WriteByte(control)
	if tagBytes != nil {
		// This is overflow tag number, if it exceeds what fits in the control byte
		buf.Write(tagBytes)
	}
	if lengthBytes != nil {
		// the number bytes did not fit into the length section of the control byte, so control byte
		// has a flag set and this has the length.
		buf.Write(lengthBytes)
	}

	var encodedVal byte
	if p.val {
		encodedVal = 0x01
	} else {
		encodedVal = 0x00
	}
	buf.WriteByte(encodedVal)
	return buf.Bytes(), nil
}

// NewContextSpecificObjectID creates an object identifier (type and instance)
func NewContextSpecificObjectID(tagNumber uint8, objectType, objectInstance uint32) (TagType, error) {
	// verify the values will fit
	if objectType&0xFC00 != 0 || objectInstance&0xFF800000 != 0 {
		return nil, bacnet.ErrInvalidData
	}
	return &ContextSpecificObjectIDType{
		ContextSpecificTypeBase: newContextSpecificTypeBase(tagNumber),
		objectType:              objectType,
		objectInstance:          objectInstance,
	}, nil

}

// NewContextSpecificObjectIDFromBytes decodes the object identifier from bytes
func NewContextSpecificObjectIDFromBytes(tagBuf *bytes.Buffer) (TagType, error) {
	control, err := tagBuf.ReadByte()
	if err != nil {
		return nil, err
	}
	tagNumber, err := decodeTagNumber(control, tagBuf)
	if err != nil {
		return nil, err
	}
	tagClass := decodeClass(control)
	if tagClass != TagContextSpecificClass {
		return nil, fmt.Errorf("Expected ContextSpecificTagClass")
	}
	tagLen, err := decodeLength(control, tagBuf)
	if err != nil {
		return nil, err
	}
	if tagLen != 4 {
		return nil, bacnet.ErrInvalidData
	}
	valBuf := make([]byte, tagLen)
	bytesRead, err := tagBuf.Read(valBuf)
	if err != nil {
		return nil, err
	} else if uint(bytesRead) != tagLen {
		return nil, bacnet.ErrInsufficientData
	}
	stuffedValue := DecodeUint(valBuf)

	return &ContextSpecificObjectIDType{
		ContextSpecificTypeBase: newContextSpecificTypeBase(tagNumber),
		objectType:              (uint32(stuffedValue) & 0xFFB00000) >> 22,
		objectInstance:          uint32(stuffedValue) & 0x003FFFFF,
	}, nil

}

// EncodeAsTagData encodes the object ID into bytes
func (p *ContextSpecificObjectIDType) EncodeAsTagData(class TagClass) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 1))

	var control byte
	tagBytes := encodeTagNumber(&control, (uint8)(p.TagNumber))
	encodeClass(&control, TagContextSpecificClass)

	lengthBytes, err := encodeLength(&control, 4)
	if err != nil {
		return nil, err
	}

	buf.WriteByte(control)
	if tagBytes != nil {
		// This is overflow tag number, if it exceeds what fits in the control byte
		buf.Write(tagBytes)
	}
	if lengthBytes != nil {
		// the number bytes did not fit into the length section of the control byte, so control byte
		// has a flag set and this has the length.
		buf.Write(lengthBytes)
	}

	// We have already validated that the values will fit in a 32 bit buffer, shift the
	shiftedType := p.objectType << 22
	stuffedVal := shiftedType | p.objectInstance
	buf.Write(EncodeUint(uint(stuffedVal), 4))
	return buf.Bytes(), nil
}
