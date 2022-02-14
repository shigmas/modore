package bacnet

// application encoding (as opposed to context specific)
type (
	APDUApplicationTag struct {
		Type       APDUType
		DataLength int
		DataValue  []byte
	}

	// APDUApplicationTypeBase has the shared encoding functions for types in the application class. The
	// specific structs implement the common interface
	APDUApplicationTypeBase struct {
		APDUTypeBase
	}

	// Rather tedious way to avoid several large and complicated switch statements
	APDUApplicationNullType struct {
		APDUApplicationTypeBase
	}

	APDUApplicationBoolType struct {
		APDUApplicationTypeBase
		val bool
	}
	APDUApplicationUnsignedIntType struct {
		APDUApplicationTypeBase
		val uint
	}
	APDUApplicationSignedIntType struct {
	}
	APDUApplicationRealType struct {
	}
	APDUApplicationDoubleType struct {
	}
	APDUApplicationOctetStringType struct {
	}
	APDUApplicationCharacterStringType struct {
	}
	APDUApplicationBitStringType struct {
	}
	APDUApplicationEnumeratedType struct {
	}
	APDUApplicationDateType struct {
	}
	APDUApplicationTimeType struct {
	}
	APDUApplicationObjectIDType struct {
	}
)

var (
	_ APDUType = (*APDUApplicationNullType)(nil)
	_ APDUType = (*APDUApplicationBoolType)(nil)
	_ APDUType = (*APDUApplicationUnsignedIntType)(nil)
)

func (p *APDUApplicationNullType) EncodeAsTagData(control *byte, class APDUTagClass) []byte {
	p.encodeTagNumber(control, uint8(APDUTagNumberDataNull))
	*control |= p.encodeClass(class)
	// NULL is just 0 for everything
	return nil
}

func (p *APDUApplicationBoolType) EncodeAsTagData(control *byte, class APDUTagClass) []byte {
	// Technically, this is not allowed, but this really clutters up the interface to have an error
	// for this one case.
	if class == APDUTagContextSpecificClass {
		// some kind of output?
	}
	p.encodeTagNumber(control, uint8(APDUTagNumberDataBool))
	*control |= p.encodeClass(class)
	shift := 0
	if !p.val {
		shift = 1
	}
	*control |= 1 << shift

	// NULL is just 0 for everything
	return nil
}
func (p *APDUApplicationUnsignedIntType) EncodeAsTagData(control *byte, class APDUTagClass) []byte {
	p.encodeTagNumber(control, uint8(APDUTagNumberDataUnsignedInt))
	*control |= p.encodeClass(class)
	return nil
}
