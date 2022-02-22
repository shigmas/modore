package apdu

// application encoding (as opposed to context specific)
type (
	ApplicationTag struct {
		Type       TagType
		DataLength int
		DataValue  []byte
	}

	// ApplicationTypeBase has the shared encoding functions for types in the application class. The
	// specific structs implement the common interface
	ApplicationTypeBase struct {
		TagTypeBase
	}

	// Rather tedious way to avoid several large and complicated switch statements
	ApplicationNullType struct {
		ApplicationTypeBase
	}

	ApplicationBoolType struct {
		ApplicationTypeBase
		val bool
	}
	ApplicationUnsignedIntType struct {
		ApplicationTypeBase
		val uint
	}
	ApplicationSignedIntType struct {
	}
	ApplicationRealType struct {
	}
	ApplicationDoubleType struct {
	}
	ApplicationOctetStringType struct {
	}
	ApplicationCharacterStringType struct {
	}
	ApplicationBitStringType struct {
	}
	ApplicationEnumeratedType struct {
	}
	ApplicationDateType struct {
	}
	ApplicationTimeType struct {
	}
	ApplicationObjectIDType struct {
	}
)

var (
	_ TagType = (*ApplicationNullType)(nil)
	_ TagType = (*ApplicationBoolType)(nil)
	_ TagType = (*ApplicationUnsignedIntType)(nil)
)

func (p *ApplicationNullType) EncodeAsTagData(class TagClass) []byte {
	var control byte
	p.encodeTagNumber(&control, uint8(TagNumberDataNull))
	p.encodeClass(&control, class)
	// NULL is just 0 for everything
	return []byte{control}
}

func (p *ApplicationBoolType) EncodeAsTagData(class TagClass) []byte {
	var control byte
	// Technically, this is not allowed, but this really clutters up the interface to have an error
	// for this one case.
	if class == TagContextSpecificClass {
		// some kind of output?
	}
	p.encodeTagNumber(&control, uint8(TagNumberDataBool))
	p.encodeClass(&control, class)
	shift := 0
	if !p.val {
		shift = 1
	}
	control |= byte(1) << shift

	// bool is encoded into the first byte
	return []byte{control}
}
func (p *ApplicationUnsignedIntType) EncodeAsTagData(class TagClass) []byte {
	var control byte
	p.encodeTagNumber(&control, uint8(TagNumberDataUnsignedInt))
	p.encodeClass(&control, class)

	// This is not right
	return []byte{control}
}
