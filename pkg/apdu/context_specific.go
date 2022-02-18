package apdu

import (
	"bytes"
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
		TagNumber int
	}

	ContextSpecificNullType struct {
		ContextSpecificTypeBase
	}

	ContextSpecificBoolType struct {
		ContextSpecificTypeBase
		val bool
	}
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
	ContextSpecificObjectIDType struct {
	}
)

func newContextSpecificTypeBase(tagNumber int) ContextSpecificTypeBase {
	return ContextSpecificTypeBase{TagNumber: tagNumber}
}

func NewContextSpecificUnsignedInt(tagNumber int, val uint) TagType {
	return &ContextSpecificUnsignedIntType{
		ContextSpecificTypeBase: newContextSpecificTypeBase(tagNumber),
		val:                     val,
	}

}

func (p ContextSpecificUnsignedIntType) EncodeAsTagData(class TagClass) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 1))

	var control byte
	tagBytes := p.encodeTagNumber(&control, (uint8)(p.TagNumber))
	p.encodeClass(&control, TagContextSpecificClass)

	length := GetUnsignedIntByteSize(p.val)
	lengthBytes := p.encodeLength(&control, length)
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
	return buf.Bytes()
}
