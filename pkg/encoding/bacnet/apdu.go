package bacnet

// APDU encoding is a bit trickier than NPDU. It's encoded with the type of service as the first byte. That will
// dictate the rest of the following bytes.
// See 20.1 in the spec (p 731 in the 2020 spec)
// Confirmed: 20.1.2 (20.1.2.11 is more clear about the bytes)
// Unconfirmed: 20.1.3 (20.1.3.3 for the bytes
//
// determines the rest of the bytes after that.
// fixed byte 0: confirmed or unconfirmed
// Unconfirmed:
// fixed byte 1: unconfirmed service ID
// remaining bytes: variable parameters.
// Confirmed:
// fixed byte 1: Maximum segments of APDU
// fixed byte 2: invoke ID
// fixed byte 3: confirmed service ID
// remaining bytes (if any): data
//
// In the encoding, the parameter is preceded by a variable length tag followed by the actual data. The tag is
// has control byte, which will indicate the type of data and some context about the parameter.

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

// TagClass indicates the bit for the class. (bit 3 in the Tag byte).
type APDUTagClass int8

const (
	APDUTagApplicationClass     = iota
	APDUTagContextSpecificClass = 1
)

type APDUTagNumberDataType uint8

const (
	APDUTagNumberDataNull            = iota // 0
	APDUTagNumberDataBool                   // 1
	APDUTagNumberDataUnsignedInt            // 2
	APDUTagNumberDataSignedInt              // 3
	APDUTagNumberDataReal                   // 4
	APDUTagNumberDataDouble                 // 5
	APDUTagNumberDataOctetString            // 6
	APDUTagNumberDataCharacterString        // 7
	APDUTagNumberDataBitString              // 8
	APDUTagNumberDataEnumerated             // 9
	APDUTagNumberDataDate                   // 10
	APDUTagNumberDataTime                   // 11
	APDUTagNumberDataObjectID               // 12
	APDUTagNumberDataReserved1              // 13
	APDUTagNumberDataRserved2               // 14
	APDUTagNumberDataReserved3              // 15
	APDUTagNumberDataApplicationTag         // 16
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
		ServiceData          []APDUParameter
	}

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
	// Constructed (context specific, I guess): (byte 3 is 1)
	//  - Length is set to 110
	//  - The encoding
	//  - In addition, there will be a Closing tag, which is identical to the opening tag, but length is 111
	//
	// (*)Tag Number: Technically, both Application and ContextSpecific tags have "tag numbers", but for
    // Application tags, the tag number is overloaded as the type, so we don't include a tag number in the
    // struct. The enum includes TagNumber, though as an indication that it goes in the tag number slot in
    // the data. Either way, it's confusing, but hopefully, less confusing than the spec.
	APDUApplicationTag struct {
		DataType   APDUTagNumberDataType
		DataLength uint64
		DataValue  []byte
	}

	APDUContexetSpecificTag struct {
		DataType   APDUTagNumberDataType
        TagNumber uint8
		DataLength uint64
		DataValue  []byte
	}

	APDUParameter struct {
		Type APDUTagClass
		data interface{}
	}
)

// NewAPDUContextSpecificTagFromUint32 creates a tag for an uint32
// Note: even though this takes a uint32, the spec says that it should be in the smallest number of bytes
// possible.
func NewAPDUContextSpecificTagFromUint32(d uint32, tagNumber uint8) {

    return &APDUContexetSpecificTag{
        DataType: 
    }
}

func (p *APDUParameter)encode() []byte {
    switch p.Type {
    case APDUTagApplicationClass:
        
    case APDUTagContextSpecificClass:
}


func NewWhoisAPDUMessage(low, high uint32) *APDUUnconfirmedMessage {
	return &APDUUnconfirmedMessage{
		APDUMessage:          APDUMessage{PDUTypeUnconfirmedServiceRequest},
		UnConfirmedServiceID: PDUServiceUnconfirmedWhoIs,
		ServiceData: []APDUParameter{
			{APDUTabContextSpecificClass, low},
			{APDUTabContextSpecificClass, high},
		},
	}

}

// generic encoding for internals
