// Package apdu is the Application Protocol Data Unit for bacnet. This is the highest level "protocol" of bacnet.
// As the name says, it provides the application level information for bacnet, so it is actually the biggest
// internal package. As it is a 'protocol', the user level interaction is done outside of this package.
// APDU messages have their own encoding, which is in this package, of course. . But, the message data is made of
// primitive and not so primitive types. Those encodings are also in this package.

package apdu

import (
	"bytes"
	"errors"

	"github.com/shigmas/modore/pkg/bacnet"
)

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
type PDUType uint8

// The values for PDUType. I'm not sure if we will handle all of these
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

// ServiceConfirmed is the type of service for confirmed requests
type ServiceConfirmed uint8

// The values for ServiceConfirmed. We are explicit because these are transmitted.
const (
	ServiceConfirmedAcknowledgeAlarm ServiceConfirmed = 0
	ServiceConfirmedCovNotofication                   = 1
	// This is just a partial list. Fill these in later
	// ...
	ServiceConfirmedMax = 30
)

// ServiceUnconfirmed do not need confirmations.
type ServiceUnconfirmed uint8

// The values for ServiceUnconfirmed. We are explicit because these are transmitted.
const (
	ServiceUnconfirmedIAm               ServiceUnconfirmed = 0
	ServiceUnconfirmedIHave                                = 1
	ServiceUnconfirmedCOVNotification                      = 2
	ServiceUnconfirmedEventNotification                    = 3
	ServiceUnconfirmedPrivateTransfer                      = 4
	ServiceUnconfirmedTextMessage                          = 5
	ServiceUnconfirmedTimeSync                             = 6
	ServiceUnconfirmedWhoHas                               = 7
	ServiceUnconfirmedWhoIs                                = 8
	ServiceUnconfirmedUTCTimeSync                          = 9
	ServiceUnconfirmedWriteGroup                           = 10
	ServiceUnconfirmedMax                                  = 11
)

type (
	// Message is the basic interface for apdu messages. Move this to a public pkg
	Message interface {
		Encode() ([]byte, error)
	}

	// MessageBase is the base type for the various types of APDU messages.
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
		MaxSegmentsAccepted       uint8 // Only up to 7
		MaxLengthAccepted         uint8 // Only up to 8
		InvokeID                  uint8
		SequenceNumber            *uint8 // if IsSegmented is true
		ProposedWindowSize        *uint8 // if IsSegmented is true
		ServiceID                 ServiceConfirmed
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
		ServiceID ServiceUnconfirmed
		//ServiceData []TagType
		ServiceData []TagType
	}
)

var (
	_ (Message) = (*ConfirmedMessage)(nil)
	_ (Message) = (*UnconfirmedMessage)(nil)
)

// NewMessageFromBytes creates an APDU message from bytes by interpreting the first byte.
func NewMessageFromBytes(data []byte) (Message, error) {
	if len(data) < 1 {
		return nil, errors.New("bytes do not contain an NPDU message")
	}
	pduType := PDUType(data[0] & 0xF0)
	switch pduType {
	case PDUTypeConfirmedServiceRequest:
		return newConfirmedMessageFromBytes(pduType, data)
	case PDUTypeUnconfirmedServiceRequest:
		return newUnconfirmedMessageFromBytes(pduType, data)
	default:
		return nil, errors.New("Unimplemented PDUType")
	}

}

func newConfirmedMessageFromBytes(pdu PDUType, data []byte) (*ConfirmedMessage, error) {
	if len(data) < 3 {
		return nil, errors.New("insufficient length for message type")
	}
	control := data[0]
	maxSegs := data[1] & 0b0111000 >> 4
	maxLen := data[1] & 0x0F

	msg := ConfirmedMessage{
		MessageBase:               MessageBase{pdu},
		IsSegmented:               (control & 0x04) != 0,
		DoSegmentsFollow:          (control & 0x03) != 0,
		IsSegmentResponseAccepted: (control & 0x02) != 0,
		MaxSegmentsAccepted:       maxSegs,
		MaxLengthAccepted:         maxLen,
		InvokeID:                  data[2],
	}
	currByteIndex := 3
	if msg.IsSegmented {
		if len(data) < 5 {
			return nil, errors.New("insufficient length for message type")
		}
		seqNumber := data[currByteIndex]
		currByteIndex++
		msg.SequenceNumber = &seqNumber
		winSize := data[currByteIndex]
		currByteIndex++
		msg.ProposedWindowSize = &winSize
	}

	msg.ServiceID = ServiceConfirmed(data[currByteIndex])
	currByteIndex++

	// These are parameters. Need to decode these too
	msg.ServiceData = data[currByteIndex:]

	return &msg, nil

}

// The first byte is the control byte, which has already been parsed, so unconfirmed messages
// read from the second byte onward
func newUnconfirmedMessageFromBytes(pdu PDUType, data []byte) (*UnconfirmedMessage, error) {
	if len(data) < 2 {
		return nil, errors.New("insufficient length for message type")
	}

	msg := UnconfirmedMessage{
		MessageBase: MessageBase{pdu},
		ServiceID:   ServiceUnconfirmed(data[1]),
	}

	// The parameters depend on the service type. So, we just have a big switch and parse the data
	buf := bytes.NewBuffer(data[2:])
	switch msg.ServiceID {
	case ServiceUnconfirmedIAm:

	case ServiceUnconfirmedWhoIs:
		lowTag, err := NewContextSpecificUnsignedIntFromBytes(buf)
		if err != nil {
			return nil, err
		}
		highTag, err := NewContextSpecificUnsignedIntFromBytes(buf)
		if err != nil {
			return nil, err
		}
		return &UnconfirmedMessage{
			MessageBase: MessageBase{PDUTypeUnconfirmedServiceRequest},
			ServiceID:   ServiceUnconfirmedWhoIs,
			ServiceData: []TagType{lowTag, highTag},
		}, nil
	default:
		return nil, bacnet.ErrNotImplemented
	}

	return &msg, nil
}

// NewWhoisMessage is just here temporarily. This should be in bacnet, but it requires that we export more types.
func NewWhoisMessage(low, high uint) (*UnconfirmedMessage, error) {
	lowTag, err := NewContextSpecificUnsignedInt(0, low)
	if err != nil {
		return nil, err
	}
	highTag, err := NewContextSpecificUnsignedInt(1, high)
	if err != nil {
		return nil, err
	}
	return &UnconfirmedMessage{
		MessageBase: MessageBase{PDUTypeUnconfirmedServiceRequest},
		ServiceID:   ServiceUnconfirmedWhoIs,
		ServiceData: []TagType{lowTag, highTag},
	}, nil

}

func NewIAmMessage(objectID, objectInstance uint32, maxAPDULengthAccepted uint, segmentationSupported bool,
	vendorID uint16) (*UnconfirmedMessage, error) {

	devID, err := NewContextSpecificObjectID(0, objectID, objectInstance)
	if err != nil {
		return nil, err
	}
	maxAccepted, err := NewContextSpecificUnsignedInt(1, maxAPDULengthAccepted)
	if err != nil {
		return nil, err
	}
	segSupported, err := NewContextSpecificBool(2, segmentationSupported)
	if err != nil {
		return nil, err
	}
	vID, err := NewContextSpecificUnsignedInt(3, uint(vendorID))
	if err != nil {
		return nil, err
	}

	return &UnconfirmedMessage{
		MessageBase: MessageBase{PDUTypeUnconfirmedServiceRequest},
		ServiceID:   ServiceUnconfirmedWhoIs,
		ServiceData: []TagType{devID, maxAccepted, segSupported, vID},
	}, nil

}

// Encode for confirmed messages is unimplemented right now
func (cm *ConfirmedMessage) Encode() ([]byte, error) {
	return nil, bacnet.ErrNotImplemented
}

// Encode is for unconfirmed messages
func (um *UnconfirmedMessage) Encode() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 2))

	buf.WriteByte(byte(um.ServiceType)) // I thought << 5
	buf.WriteByte(byte(um.ServiceID))

	for _, param := range um.ServiceData {
		// This is probably more complicated than it needs to be. But, until I study the encodings for
		// each type and class, it's better to keep them separate for now.
		bs, err := param.EncodeAsTagData(TagContextSpecificClass)
		if err != nil {
			return nil, err
		}
		buf.Write(bs)
	}

	return buf.Bytes(), nil
}
