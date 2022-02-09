package bacnet

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// NetworkMessagePriority is the 2 bit priority part of the Control Information
type NetworkMessagePriority uint8 // smallest int. 6 extra bits

// NetworkMessagePriority values
const (
	NormalMessage            NetworkMessagePriority = 0b00
	UrgentMessage                                   = 0b01
	CriticalEquipmentMessage                        = 0b10
	LifeSafetyMessage                               = 0b11
)

const (
	// DefaultProtocolVersion is the NPDU protocol version, which is 1.
	DefaultProtocolVersion uint8 = 1
)

type (
	// BACnet NPDU (Network Protocol Data Unit) is a byte level protocol over the network. They are either
	// one, two, or variable bytes in length. To avoid using a 2 byte slice, we use a uint16 for the two
	// byte values, so we will use uint8 for the one byte values. (Admittedly, this might cause some confusion
	// if it's just raw data or a meaningful number. But, in BACnet, that is often the case.) For variable
	// bytes, we just use []byte.

	// NPDUControl has the following format
	// 01234567
	// 01: Network Priority
	//   2: Reply parameter
	//    3: Source specifier: About the Source
	//     4: Reserved. 0
	//      5: Destination specifier: About destination and hop
	//       6: Reserved. 0
	//        7: Flag for Network message (has message type) or application (APDU) message (no message type)
	// This is the interpretation of the second byte in the BACnet NPDU message
	// The member names are quite long, yet some of them are still unclear.
	// IsBACNetConfirmedRequestPDUPresent: if this is false, it means another type of reply is present (not
	//    that there is no reply)
	// ServiceSpecifiesPresent: SNET, SLEN, and SADR values must be set
	// DestinationSpecifiersPresent: DNET, DLEN, and DADR values must be set, as well as Hop Count.
	// IsNDSUNetworkLayerMessage: Network Service Data Unit. If false, it is a BACnet APDU (Application
	//    Protocol Data Unit)
	NPDUControl struct {
		Priority                           NetworkMessagePriority
		IsBACNetConfirmedRequestPDUPresent bool
		SourceSpecifiersPresent            bool
		DestinationSpecifiersPresent       bool
		IsNDSUNetworkLayerMessage          bool
	}

	// Specifics is the address information for Source and Destination, although the values differ slightly.
	Specifics struct {
		Network uint16
		Length  uint8  // For destination, this can be 0, which means broadcast. So Address is nil...?
		Address []byte // This could be a subnet and node, but we don't really care at this level
	}

	// NPDUMessage is the literal struct to send across the wire. This struct will be encoded and decoded off the
	// wire. Network layer Protocol Data Unit.
	// Only the ProtocolVersion and Control are guaranteed. For the other members, they are dependent on
	// the control values. Although, I think there are no empty messages.
	NPDUMessage struct {
		ProtocolVersion byte        // Version, which is probably 1
		Control         NPDUControl // Information about the rest of the struct
		Destination     *Specifics  // if this exists, HopCount should be set, but after Source
		Source          *Specifics
		HopCount        *uint8
		MessageType     *uint8
		VendorID        *uint16
		APDU            []byte // The application layer should have already encoded this
	}
)

// NewNPDUMessage creates an NPDUMessage. Depending on the control information, different portions of the
// message will be valid and others will be nil. This is kind of low level, and numerous other constructors can be
// made
func NewNPDUMessage(control NPDUControl, dest, src *Specifics, hopCount *uint8,
	messageType *uint8, vendorID *uint16, apdu []byte) *NPDUMessage {
	return &NPDUMessage{
		ProtocolVersion: DefaultProtocolVersion,
		Control:         control,
		Destination:     dest,
		Source:          src,
		HopCount:        hopCount,
		MessageType:     messageType,
		VendorID:        vendorID,
		APDU:            apdu,
	}
}

// NewNPDUControl creates a control byte from the values
func NewNPDUControl(priority NetworkMessagePriority, isBACNetConfirmedRequestPDUPresent, sourceSpecifiersPresent, destinationSpecifiersPresent, isNDSUNetworkLayerMessage bool) *NPDUControl {
	return &NPDUControl{
		Priority:                           priority,
		IsBACNetConfirmedRequestPDUPresent: isBACNetConfirmedRequestPDUPresent,
		SourceSpecifiersPresent:            sourceSpecifiersPresent,
		DestinationSpecifiersPresent:       destinationSpecifiersPresent,
		IsNDSUNetworkLayerMessage:          isNDSUNetworkLayerMessage,
	}
}

// Add this method to byte.Buffer for our usage. I actually don't know if it's big or little endian yet, so this
// is to encapsulate that.
func readDoubleByte(buf *bytes.Buffer) (uint16, error) {
	b := make([]byte, 2)
	count, err := buf.Read(b)
	if count != 2 {
		return 0, fmt.Errorf("Did not read 2 bytes for uint16")
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

func writeDoubleByte(buf *bytes.Buffer, db uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, db)
	count, err := buf.Write(b)
	if count != 2 {
		return fmt.Errorf("Did not write 2 bytes for uint16")
	}
	return err
}

// XXX - Do I have the bits backwards?
func encodeNPDUControl(ctrl NPDUControl) byte {
	// Start from the most significant bytes, and shift
	var encoded byte
	if ctrl.IsNDSUNetworkLayerMessage {
		encoded |= 1
	}
	// shift two because reserved bit
	encoded = encoded << 2
	if ctrl.DestinationSpecifiersPresent {
		encoded |= 1
	}
	// shift two because reserved bit
	encoded = encoded << 2
	if ctrl.SourceSpecifiersPresent {
		encoded |= 1
	}
	encoded = encoded << 1
	if ctrl.IsBACNetConfirmedRequestPDUPresent {
		encoded |= 1
	}
	// next one is two bits
	encoded = encoded << 2
	encoded |= (uint8)(ctrl.Priority)

	return encoded
}

// XXX - Do I have the bits backwards?
func decodeNPDUControl(data byte) NPDUControl {
	// Start from the most significant bytes, and shift
	var ctrl NPDUControl
	ctrl.IsNDSUNetworkLayerMessage = (data & 0b10000000) != 0
	ctrl.DestinationSpecifiersPresent = (data & 0b00100000) != 0
	ctrl.SourceSpecifiersPresent = (data & 0b00001000) != 0
	ctrl.IsBACNetConfirmedRequestPDUPresent = (data & 0b00000100) != 0
	ctrl.Priority = (NetworkMessagePriority)(data & 0b00000011)

	return ctrl
}

func writeSpecifics(buf *bytes.Buffer, spec *Specifics) error {
	if e := writeDoubleByte(buf, spec.Network); e != nil {
		return e
	}
	if e := buf.WriteByte(spec.Length); e != nil {
		return e
	}
	if spec.Length > 0 {
		// Do we need this? Otherwise, maybe we pass in nil to ByteBuffer.Write?
		if _, e := buf.Write(spec.Address); e != nil {
			return e
		}
	}
	return nil
}

func readSpecifics(buf *bytes.Buffer) (*Specifics, error) {
	var spec Specifics
	db, e := readDoubleByte(buf)
	if e != nil {
		return nil, e
	}
	spec.Network = db
	b, e := buf.ReadByte()
	if e != nil {
		return nil, e
	}
	spec.Length = b
	if spec.Length > 0 {
		// read uses len, not capacity
		addr := make([]byte, spec.Length)
		bytesRead, e := buf.Read(addr)
		if e != nil {
			return nil, e
		}
		if bytesRead != int(spec.Length) {
			return nil, fmt.Errorf("read %d bytes, expected %d bytes", bytesRead, spec.Length)
		}
		spec.Address = addr
	}
	return &spec, nil
}

// Encode an NPDUMessage
func Encode(message *NPDUMessage) ([]byte, error) {
	// We only know that it will be 2 bytes plus data.
	b := make([]byte, 0, 3)
	buf := bytes.NewBuffer(b)
	if e := buf.WriteByte(message.ProtocolVersion); e != nil {
		return nil, e
	}
	if e := buf.WriteByte(encodeNPDUControl(message.Control)); e != nil {
		return nil, e
	}

	if message.Control.DestinationSpecifiersPresent {
		if e := writeSpecifics(buf, message.Destination); e != nil {
			return nil, e
		}
	}
	if message.Control.SourceSpecifiersPresent {
		if e := writeSpecifics(buf, message.Source); e != nil {
			return nil, e
		}
	}
	if message.Control.DestinationSpecifiersPresent {
		if e := buf.WriteByte(*message.HopCount); e != nil {
			return nil, e
		}
	}
	// vendorID goes in between message type and APDU, but I don't which one it accompanies.
	if message.Control.IsNDSUNetworkLayerMessage {
		if e := buf.WriteByte(*message.MessageType); e != nil {
			return nil, e
		}
	} else {
		if _, e := buf.Write(message.APDU); e != nil {
			return nil, e
		}
	}

	return buf.Bytes(), nil

}
