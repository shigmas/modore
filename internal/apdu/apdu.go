package apdu

import "bytes"

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
	PDUTypeConfirmedServiceRequest   PDUType = 0
	PDUTypeUnconfirmedServiceRequest         = 0x10
	PDUTypeComplexAck                        = 0x30
	PDUTypeSegmentAck                        = 0x40
	PDUTypeError                             = 0x50
	PDUTypeReject                            = 0x60
	PDUTypeAbort                             = 0x70

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

// ServiceUnconfirmed do not need confirmations. Should just be service, and we can figure out
// confirmed/unconfirmed, since it can't be both.
type ServiceUnconfirmed uint8

// The values for ServiceUnconfirmed
const (
	ServiceUnconfirmedIAm ServiceUnconfirmed = iota
	ServiceUnconfirmedIHave
	ServiceUnconfirmedCOVNotification
	ServiceUnconfirmedEventNotification
	ServiceUnconfirmedPrivateTransfer
	ServiceUnconfirmedTextMessage
	ServiceUnconfirmedTimeSync
	ServiceUnconfirmedWhoHas
	ServiceUnconfirmedWhoIs
	ServiceUnconfirmedUTCTimeSync
	ServiceUnconfirmedWriteGroup
)

type (
	Message interface {
		Encode() ([]byte, error)
	}

	// Message is the base type for the various types of APDU messages.
	MessageBase struct {
		// The first nibble of the message is the service type. That's the only common part of the messages
		ServiceType PDUType
	}

	// ConfirmedMessage has the following encoding:
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
	//  |      .                        |
	//  |      .                        |
	//  |---|---|---|---|---|---|---|---|
	// Implementation note: We use pointers to show that parts of the message may be empty. It's not a
	// good way, but it's convenient shorthand.
	ConfirmedMessage struct {
		MessageBase
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

	// UnconfirmedMessage is a little simpler and has the following encoding:
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
	// Specifically, Meaning that the other flags that may be set in the ConfirmedMessage are all 0.
	UnconfirmedMessage struct {
		MessageBase
		// The rest of the byte for the PDUType is 0
		UnConfirmedServiceID PDUService
		ServiceData          []TagType
	}
)

var (
	_ (Message) = (*ConfirmedMessage)(nil)
	_ (Message) = (*UnconfirmedMessage)(nil)
)

// signed matters, size does not, since we need to make it as small as possible in encoding.
func NewWhoisMessage(low, high uint) *UnconfirmedMessage {
	return &UnconfirmedMessage{
		MessageBase:          MessageBase{PDUTypeUnconfirmedServiceRequest},
		UnConfirmedServiceID: PDUServiceUnconfirmedWhoIs,
		ServiceData: []TagType{
			NewContextSpecificUnsignedInt(0, low),
			NewContextSpecificUnsignedInt(1, high),
		},
	}
}

func (cm *ConfirmedMessage) Encode() ([]byte, error) {
	return nil, nil
}

// This is generic enough to encode all Unconfirmed messages.
func (um *UnconfirmedMessage) Encode() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 2))

	buf.WriteByte(byte(um.ServiceType)) // I thought << 5
	buf.WriteByte(byte(um.UnConfirmedServiceID))

	for _, param := range um.ServiceData {
		// This is probably more complicated than it needs to be. But, until I study the encodings for
		// each type and class, it's better to keep them separate for now.
		buf.Write(param.EncodeAsTagData(TagContextSpecificClass))
	}

	return buf.Bytes(), nil
}
