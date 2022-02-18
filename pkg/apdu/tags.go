package apdu

// Message Parameters
// These are base types or interfaces. Concrete implementations are in the respective files.
// APDU tags: all about encoding APDU message parameters.
// APDU messages have parameters in the request data, Described in 20.2 of the spec. There are two
// types of parameters. The parameters are identified by tags, which fall into two "classes":
// fundamental data types (application tags), and inferred types (context specific tags). As with most
// other encoded BACnet types, the first byte describes the rest of the tag, including the length.
// Bit Number:
//    7     6     5     4     3     2     1     0
// |-----|-----|-----|-----|-----|-----|-----|-----|
// | Tag Number(*)         |Class|Length/Value/Type|
// |-----|-----|-----|-----|-----|-----|-----|-----|
// Tag Number: tags from 0-14 will be in the first four bits. 1111 (15) is a flag indicating that
//    the tag number is in the immediate byte after this one, up to 254. >255 is not allowed. For
//    Application data, the tag number indicates the type of data (bool, int, etc.). For Context
//    Specific, it is... context specific.
// Class: 0: application, 1: context specific
// Length/Value/Type: This might be the actual data, or it might just be describing the data, depending
// on the class and the data itself (20.2.1.3.1 in the spec)
// Application (byte 3 is 0)
//  - Boolean: The three remaining bits of the byte are: False: 000, True: 001
//  - Others:
//    Length:
//    0 <=  4:  Bits 2-0, with 2 being the most significant. (So, 000-100)
//    5 <= 253: Bits 2-0 set to 101, and the next byte is the length. unless the Tag Number > 14, it
//              is the byte after the tag number byte.
//    254 <= 65535: Bits 2-0 set to 101, and the next byte is 0xFE, unless the Tag Number > 14, it
//              is the byte after the tag number byte, and the next *two* bytes are the length
//    65536 <= 2^32-1: Bits 2-0 set to 101, and the next byte is 0xFF, unless the Tag Number > 14, it
//              is the byte after the tag number byte, and the next *four* bytes are the length
//     > 2^32-1 are not allowed
//
// Constructed Context specific (byte 3 is 1):
//  - Tag number is arbitrary (context specific)
//  - Length: this is always the length. Data will be stored outside of this byte.
// Constructed (context specific, I guess):
//  - Length is always length (no encoding boolean or small integers in this space)
//  - In addition, there will be a Closing tag, which is identical to the opening tag, but length is 111
//
// (*)Tag Number: Technically, both Application and ContextSpecific tags have "tag numbers", but for
// Application tags, the tag number is overloaded as the type, so we don't include a tag number in the
// struct. The enum includes TagNumber, though as an indication that it goes in the tag number slot in
// the data. Either way, it's confusing, but hopefully, less confusing than the spec.
// TagClass indicates the bit for the class. (bit 3 in the Tag byte).

type TagClass int8

const (
	TagApplicationClass     = iota
	TagContextSpecificClass = 1
)

// TagNumberType is used in to specify the type in the tags, but is only in the tag number field for
// application class tags.
type TagNumberType uint8

const (
	TagNumberDataNull            TagNumberType = iota // 0
	TagNumberDataBool                                 // 1
	TagNumberDataUnsignedInt                          // 2
	TagNumberDataSignedInt                            // 3
	TagNumberDataReal                                 // 4
	TagNumberDataDouble                               // 5
	TagNumberDataOctetString                          // 6
	TagNumberDataCharacterString                      // 7
	TagNumberDataBitString                            // 8
	TagNumberDataEnumerated                           // 9
	TagNumberDataDate                                 // 10
	TagNumberDataTime                                 // 11
	TagNumberDataObjectID                             // 12
	TagNumberDataReserved1                            // 13
	TagNumberDataRserved2                             // 14
	TagNumberDataReserved3                            // 15
	TagNumberDataApplicationTag                       // 16
)

type (
	TagType interface {
		// Encodes the type into the control byte and the data bytes if necessary. For null or bool, for
		// example, the return value is nil and the data is encoded into the control byte.
		// This function will clear the existing control
		// XXX: Pass in a byte Buffer for efficiency. In this case, tags are part of an APDU. In fact,
		// maybe all encodings can take a byte.Buffer. That way, everything is in one buffer
		EncodeAsTagData(class TagClass) []byte
	}

	// Base for all types
	TagTypeBase struct {
	}
)

// Functions for parameters common to application and context specific

// For application parameters/tags, they are under 14 and will fit in the control byte. For context specific,
// they *can* be larger than 14, the nibble is set to F and the tag is first byte in the slice. Tag numbers
// larger than 255 are not supported.
func (b *TagTypeBase) encodeTagNumber(control *byte, tagNumber uint8) []byte {
	if tagNumber < 14 {
		*control = byte(tagNumber << 4)
		return nil
	} else {
		*control = byte(0xF0 | *control)
		tagBytes := make([]byte, 1)
		tagBytes[0] = tagNumber
		return tagBytes
	}

}

func (b *TagTypeBase) encodeClass(control *byte, t TagClass) {
	// should | this like the other functions
	*control |= byte(t << 3)
}

func (b *TagTypeBase) encodeLength(control *byte, len uint) []byte {

	// I'm not sure if these comments help explain it better than the code. But, at least there are
	// 2 explanations
	// 0 <= len <= 4:
	// control: | 0-4|class|len|
	//
	// 5 <= len <= 253
	// control: | 0-4|class|101| (bytes 5 and 7 are set, 6 is not){
	// len byte 0: length

	// 254 <= len <= 65535
	// control: | 0-4|class|101| (bytes 5 and 7 are set, 6 is not){
	// len byte 0: 254
	// len byte 1,2: length
	//
	// 65535 <= len <= 2^32-1
	// control: | 0-4|class|101| (bytes 5 and 7 are set, 6 is not){
	// len byte 0: 255
	// len byte 1-4: length
	var lengthBytes []byte
	if len <= 4 {
		*control |= byte(len)
	} else {
		*control |= 5
		if len <= 253 { // up to 253, length is the first byte, and the data is following
			lengthBytes = EncodeUint(len, 1)
		} else if len <= 65535 {
			lengthBytes = make([]byte, 1, 1)
			lengthBytes[0] = 254
			lengthBytes = append(lengthBytes, EncodeUint(len, 2)...)
		} else {
			lengthBytes[0] = 255
			lengthBytes = append(lengthBytes, EncodeUint(len, 4)...)
		}
	}

	return lengthBytes
}

// Encoding and decoding helpers

// GetUnsignedIntByteSize returns the number of bytes the int will take. If an uint64 will fit into 3 bytes,
// we will encoded to fit in 3 bytes.
func GetUnsignedIntByteSize(val uint) uint {
	if val <= 0xFF {
		return 1
	} else if val <= 0xFFFF {
		return 2
	} else if val <= 0xFFFFFF {
		return 3
	} else if val <= 0x00000000FFFFFFFF {
		return 4
	} else if val <= 0x000000FFFFFFFFFF {
		return 5
	} else if val <= 0x0000FFFFFFFFFFFF {
		return 6
	} else if val <= 0x00FFFFFFFFFFFFFF {
		return 7
	} else {
		return 8
	}
}

// encoding/binary is preferable than doing this ourselves, but it doesn't work for 2 reasons:
// 1. if we have a 64 bit integer that is small enough to fit into 8 bits, BACnet wants the 8 bit integer
// 2. BACnet also wants 3
// why would they?), but BACnet wants

// EncodeUint encodes the data into a byte array
func EncodeUint(val uint, numBytes uint) []byte {

	buf := make([]byte, numBytes, numBytes)
	var i uint
	for i = 0; i < numBytes; i++ {
		shift := (numBytes - 1 - i) * 8
		// mask all but one byte of the val and set each element of the byte array
		mask := uint(0xFF << shift)
		buf[i] = (byte)((val & mask) >> shift)
	}

	return buf
}

// DecodeUint takes the raw byte array and converts it back to the Uint
func DecodeUint(raw []byte) uint {

	var val uint
	numBytes := len(raw)
	for i := 0; i < numBytes; i++ {
		shift := (numBytes - 1 - i) * 8
		bVal := (uint)(raw[i]) << shift
		val += bVal
	}

	return val
}
