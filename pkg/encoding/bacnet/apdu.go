package bacnet

// Summary: (For more detail, the actual bits are laid out in front the struct that are represented by the bits
// APDU encoding is a bit trickier than NPDU. There are eight types of message request types: Confirmed,
// Unconfirmed, and 6 others (see the spec or code). The first byte specifies the message type, and the
// contents of the following bytes. The length is depends on the type of message request:
// fixed byte 0: confirmed or unconfirmed
// Unconfirmed:
// fixed byte 1: unconfirmed service ID
// remaining bytes: variable parameters.
// Confirmed:
// fixed byte 1: Maximum segments of APDU
// fixed byte 2: invoke ID
// fixed byte 3: confirmed service ID
// remaining bytes (if any): data
// See APDUMessage for details
//
// The data consists of parameters where the 5th (or 3rd, depending on the direction you start) determines
// the type of data. So, there are only 2 types. The meaning of the rest of the data of the first byte (and of
// there are subsequent bytes) are determined by the 5th bit and the contents of the rest of the byte).
// The two different types of encoding are: application and context specific.
// The types of request are the first part of the encoding, and that determines the rest of the bytes:
// See 20.1 in the spec (p 731 in the 2020 spec)
// Confirmed: 20.1.2 (20.1.2.11 is more clear about the bytes)
// Unconfirmed: 20.1.3 (20.1.3.3 for the bytes
// Since this is quite a bit more complicated, the encodings are in separate files: application.go and
// context_specific.go.
// Note: If you're looking at the bacnet code, there's a #ifdef in bacint.h that typedef's
// BACNET_UNSIGNED_INTEGER to uint32 or uint64. I think we can just assume our platform, so we don't need to
// deal with that. But, maybe.

// PDUType is at least unconfirmed or confirmed, but more as well.
type PDUType int8

const (
	// The bacnet code shifts these instead of incrementing. We can do that if it makes sense.
	PDUTypeConfirmedServiceRequest   PDUType = 0
	PDUTypeUnconfirmedServiceRequest         = 1
	// 6 more
)

// PDUService is the base service type
type PDUService int8

// TBD: Make two "subclasses" PDUUnconfirmedService and PDUConfirmedService. Or, a template type
const (
	PDUServiceUnconfirmedIAm PDUService = iota
	// Need to fill these in, since I think these are the values sent over the wire
	PDUServiceUnconfirmedWhoIs = 8
)

type (
	// APDUMessage is the base type for the various types of APDU messages.
	APDUMessage struct {
		// The first nibble of the message is the service type. That's the only common part of the messages
		ServiceType PDUType
	}
	//    7   6   5   4   3   2   1   0
	//  |---|---|---|---|---|---|---|---|
	//  | PDU Type      |SEG|MOR| SA| 0 |
	//  |---|---|---|---|---|---|---|---|
	//  | 0 | Max Segs  | Max Resp      |
	//  |---|---|---|---|---|---|---|---|
	//  | Invoke ID                     |
	//  |---|---|---|---|---|---|---|---|
	//  | Sequence Number               | Only present if SEG = 1
	//  |---|---|---|---|---|---|---|---|
	//  | Proposed Window Size          | Only present if SEG = 1
	//  |---|---|---|---|---|---|---|---|
	//  | Service Choice                |
	//  |---|---|---|---|---|---|---|---|
	//  | Service Request               |
	//  |      .                        |
	//         .
	//  |      .                        |
	//  |---|---|---|---|---|---|---|---|
	APDUConfirmedMessage struct {
		APDUMessage
		// 3 bits of the second half of the nibble. (last bit is 0)
		IsSegmented               bool
		DoSegmentsFollow          bool
		IsSegmentResponseAccepted bool
		InvokeID                  uint8
		SequenceNumber            *uint8 // if IsSegmented is true
		ProposedWindowSize        *uint8 // if IsSegmented is true
		ConfirmedServiceID        uint8
		ServiceData               []byte
	}
	//   7   6   5   4   3   2   1   0
	// |---|---|---|---|---|---|---|---|
	// | PDU Type      | 0 | 0 | 0 | 0 |
	// |---|---|---|---|---|---|---|---|
	// | Service Choice                |
	// |---|---|---|---|---|---|---|---|
	// | Service Request               |
	// |     .                         |
	//       .
	// |     .                         |
	// |---|---|---|---|---|---|---|---|
	APDUUnconfirmedMessage struct {
		APDUMessage
		// The rest of the byte for the PDUType is 0
		UnConfirmedServiceID PDUService
		ServiceData          []APDUType
	}
)

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

type APDUTagClass int8

const (
	APDUTagApplicationClass     = iota
	APDUTagContextSpecificClass = 1
)

// APDUTagNumberType is used in to specify the type in the tags, but is only in the tag number field for
// application class tags.
type APDUTagNumberType uint8

const (
	APDUTagNumberDataNull            APDUTagNumberType = iota // 0
	APDUTagNumberDataBool                                     // 1
	APDUTagNumberDataUnsignedInt                              // 2
	APDUTagNumberDataSignedInt                                // 3
	APDUTagNumberDataReal                                     // 4
	APDUTagNumberDataDouble                                   // 5
	APDUTagNumberDataOctetString                              // 6
	APDUTagNumberDataCharacterString                          // 7
	APDUTagNumberDataBitString                                // 8
	APDUTagNumberDataEnumerated                               // 9
	APDUTagNumberDataDate                                     // 10
	APDUTagNumberDataTime                                     // 11
	APDUTagNumberDataObjectID                                 // 12
	APDUTagNumberDataReserved1                                // 13
	APDUTagNumberDataRserved2                                 // 14
	APDUTagNumberDataReserved3                                // 15
	APDUTagNumberDataApplicationTag                           // 16
)

type (
	APDUType interface {
		// Encodes the type into the control byte and the data bytes if necessary. For null or bool, for
		// example, the return value is nil and the data is encoded into the control byte.
		// This function will clear the existing control
		EncodeAsTagData(control *byte, class APDUTagClass) []byte
	}

	// Base for all types
	APDUTypeBase struct {
	}
)

// NewAPDUContextSpecificTagFromUint32 creates a tag for an uint32
// Note: even though this takes a uint32, the spec says that it should be in the smallest number of bytes
// possible.
// func NewAPDUContextSpecificTagFromUint(d uint, tagNumber uint8) *APDUContextSpecificTag {

// 	return &APDUContextSpecificTag{
// 		APDUParameterBase: APDUParameterBase{
// 			Type:     APDUTagContextSpecificClass,
// 			DataType: APDUTagNumberDataUnsignedInt,
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

// signed matters, size does not, since we need to make it as small as possible in encoding.
func NewWhoisAPDUMessage(low, high uint) *APDUUnconfirmedMessage {
	return &APDUUnconfirmedMessage{
		APDUMessage:          APDUMessage{PDUTypeUnconfirmedServiceRequest},
		UnConfirmedServiceID: PDUServiceUnconfirmedWhoIs,
		ServiceData: []APDUType{
			NewAPDUContextSpecificUnsignedInt(low, 0),
			NewAPDUContextSpecificUnsignedInt(high, 1),
		},
	}

}

// Functions for paramters common to application and context specific

// For application parameters/tags, they are under 14 and will fit in the control byte. For context specific,
// they *can* be larger than 14, the nibble is set to F and the tag is first byte in the slice. Tag numbers
// larger than 255 are not supported.
func (b *APDUTypeBase) encodeTagNumber(control *byte, tagNumber uint8) []byte {
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

func (b *APDUTypeBase) encodeClass(t APDUTagClass) byte {
	// should | this like the other functions
	return byte(t << 3)
}

func (b *APDUTypeBase) encodeLength(control *byte, len uint) []byte {

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
