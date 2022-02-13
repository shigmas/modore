package bacnet

// context specific encoding (as opposed to application)
type (
	APDUContextSpecificTypeBase struct {
		APDUTypeBase
		TagNumber int
	}

	APDUContextSpecificNullType struct {
		APDUContextSpecificTypeBase
	}

	APDUContextSpecificBoolType struct {
		APDUContextSpecificTypeBase
		val bool
	}
	APDUContextSpecificUnsignedIntType struct {
		APDUContextSpecificTypeBase
		val uint
	}
	APDUContextSpecificSignedIntType struct {
	}
	APDUContextSpecificRealType struct {
	}
	APDUContextSpecificDoubleType struct {
	}
	APDUContextSpecificOctetStringType struct {
	}
	APDUContextSpecificCharacterStringType struct {
	}
	APDUContextSpecificBitStringType struct {
	}
	APDUContextSpecificEnumeratedType struct {
	}
	APDUContextSpecificDateType struct {
	}
	APDUContextSpecificTimeType struct {
	}
	APDUContextSpecificObjectIDType struct {
	}
)

func newAPDUContextSpecificTypeBase(tagNumber int) APDUContextSpecificTypeBase {
	return APDUContextSpecificTypeBase{TagNumber: tagNumber}
}

func NewAPDUContextSpecificUnsignedInt(val uint, tagNumber int) APDUType {
	return &APDUContextSpecificUnsignedIntType{
		APDUContextSpecificTypeBase: newAPDUContextSpecificTypeBase(tagNumber),
		val:                         val,
	}

}

func (p APDUContextSpecificUnsignedIntType) EncodeAsTagData(control *byte, class APDUTagClass) []byte {
	tagBytes := p.encodeTagNumber(control, (uint8)(p.TagNumber))
	// Fix this for uniformity
	*control |= p.encodeClass(APDUTagContextSpecificClass)

	length := GetUnsignedIntByteSize(p.val)
	lengthBytes := p.encodeLength(control, length)
	encodedVal := EncodeUint(p.val, length)

	// Put it all together
	// Is the more efficient for append?
	encoded := make([]byte, 0, 1+len(tagBytes)+len(lengthBytes)+len(encodedVal))
	encoded[0] = *control
	encoded = append(encoded, tagBytes...)
	encoded = append(encoded, lengthBytes...)
	encoded = append(encoded, encodedVal...)
	return encoded
}
