package npdu

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/shigmas/modore/pkg/apdu"
)

const (
	// DefaultProtocolVersion is the NPDU protocol version, which is 1.
	DefaultProtocolVersion uint8 = 1
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

// NetworkLayerMessageType is present if bit 7 of the Control byte is set.
type NetworkLayerMessageType uint8

// NetworkLayerMessageType values
const (
	NetworkLayerWhoIsMessage                     = 0x00
	NetworkLayerIAmyMessage                      = 0x01
	NetworkLayerICouldBeMessage                  = 0x02
	NetworkLayerRejectMessage                    = 0x03
	NetworkLayerRouterBusyMessage                = 0x04
	NetworkLayerRouterAvailableMessage           = 0x05
	NetworkLayerInitializeRoutingTableMessage    = 0x06
	NetworkLayerInitializeRoutingTableAckMessage = 0x07
	NetworkLayerEstablishConnectionMessage       = 0x08
	NetworkLayerDisconnectConnectionMessage      = 0x09
	NetworkLayerWhatIsNetworkNumberMessage       = 0x12
	NetworkLayerNetworkNumberIsMessage           = 0x13
	// X'14' to X'7F': Reserved for use by ASHRAE
	// X'80' to X'FF': Available for vendor proprietary messages
)

type (
	// BACnet NPDU (Network Protocol Data Unit) is a byte level protocol over the network. They are either
	// one, two, or variable bytes in length. To avoid using a 2 byte slice, we use a uint16 for the two
	// byte values, so we will use uint8 for the one byte values. (Admittedly, this might cause some confusion
	// if it's just raw data or a meaningful number. But, in BACnet, that is often the case.) For variable
	// bytes, we just use []byte.

	// Control has the following format
	// This is the interpretation of the second byte in the BACnet NPDU message
	// The member names are quite long, yet some of them are still unclear.
	// IsBACNetConfirmedRequestPDUPresent: if this is false, it means another type of reply is present (not
	//    that there is no reply)
	// ServiceSpecifiesPresent: SNET, SLEN, and SADR values must be set
	// DestinationSpecifiersPresent: DNET, DLEN, and DADR values must be set, as well as Hop Count.
	// IsNDSUNetworkLayerMessage: Network Service Data Unit. If false, it is a BACnet APDU (Application
	//    Protocol Data Unit)
	// This will be encoded as a byte
	//   7   6   5   4   3   2   1   0
	// |---|---|---|---|---|---|---|---|
	// | N | R | D | R | S | C | Prio  |
	// N: 1: Network layer, 0: APDU (no message type - see message struct)
	// bits 6 and 4 are Reserved: 0
	// D: Destination Address specifier 0: not present, 1: Present
	// S: Source Address specifier 0: not present, 1: Present
	// C: flag for Confirmed request PDU
	// Priority: See NetworkMessagePriority
	Control struct {
		Priority                           NetworkMessagePriority
		IsBACNetConfirmedRequestPDUPresent bool
		SourceAddressPresent               bool
		DestinationAddressPresent          bool
		IsNDSUNetworkLayerMessage          bool
	}

	// Address is the address information for Source and Destination, although the values differ slightly.
	// MacLen 0 means broacdast, Length 0 means broadcast Mac, and Address is nil.
	// These are defined in 6.2.2, but the encoding is different for each type. We are only supporting
	// "BACnet/IP", which is essentially IP addresses. Addresses are 6 bytes long: From the most
	// significant to the least: the IP address, then 2 bytes for the port
	Address struct {
		// The source code has Mac and MacLen, but the encoding does not. I think it may be for
		// creating addresses, but I don't think we need them.
		// MacLen  uint8  // MacLen, but really a flag. ([]byte has len that we use for encoding)
		// Mac     []byte // for IP, 4 bytes for IP, 2 for port. I guess it only supports v4
		// Network is 1-65535 for Destinations, and 1-65534 for Sources
		// In any case, I don't think we'll have both Mac and Address set, so we only need one.
		Network uint16
		Length  uint8  // It's called Length, but it's really flag. Rename this.
		Addr    []byte // This could be a subnet and node, but we don't really care at this level
	}

	// Message is the literal struct to send across the wire. This struct will be encoded and decoded off
	// the wire. This is the NPDU in the spec (6.2) Network layer Protocol Data Unit.
	// Only the ProtocolVersion and Control are guaranteed. For the other members, they are dependent on
	// the control values. Although, I think there are no empty messages. Unless it is an actual byte,
	// uint8 will be preferred over byte.
	// Implementation note: We use pointers to show that parts of the message may be empty. It's not a
	// good way, but it's convenient shorthand.
	Message struct {
		ProtocolVersion uint8    // Version, which is probably 1
		Control         Control  // Information about the rest of the struct
		Destination     *Address // if this exists, HopCount should be set, but after Source
		Source          *Address
		HopCount        *uint8
		MessageType     NetworkLayerMessageType // enum, so can't be nil, and not good to make it uint8
		VendorID        *uint16
		APDU            apdu.Message
	}
)

// NewControl creates a control byte from the values. We return the actual struct instead of a pointer because
// that is how it will generally be used.
func NewControl(priority NetworkMessagePriority, isBACNetConfirmedRequestPDUPresent, sourceAddressPresent,
	destinationAddressPresent, isNDSUNetworkLayerMessage bool) Control {
	return Control{
		Priority:                           priority,
		IsBACNetConfirmedRequestPDUPresent: isBACNetConfirmedRequestPDUPresent,
		SourceAddressPresent:               sourceAddressPresent,
		DestinationAddressPresent:          destinationAddressPresent,
		IsNDSUNetworkLayerMessage:          isNDSUNetworkLayerMessage,
	}
}

// NewMessage creates an NPDUMessage. Depending on the control information, different portions of the
// message will be valid and others will be nil. This is kind of low level, and numerous other
// constructors can be made
func NewMessage(control Control, dest, src *Address, hopCount uint8,
	messageType NetworkLayerMessageType, vendorID *uint16, apdu apdu.Message) *Message {
	return &Message{
		ProtocolVersion: DefaultProtocolVersion,
		Control:         control,
		Destination:     dest,
		Source:          src,
		HopCount:        &hopCount,
		MessageType:     messageType,
		VendorID:        vendorID,
		APDU:            apdu,
	}
}

//func NewAddress(

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
func encodeControl(ctrl Control) byte {
	// Start from the most significant bytes, and shift
	var encoded byte
	if ctrl.IsNDSUNetworkLayerMessage {
		encoded |= 1
	}
	// shift two because reserved bit
	encoded = encoded << 2
	if ctrl.DestinationAddressPresent {
		encoded |= 1
	}
	// shift two because reserved bit
	encoded = encoded << 2
	if ctrl.SourceAddressPresent {
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
func decodeControl(data byte) Control {
	// Start from the most significant bytes, and shift
	var ctrl Control
	ctrl.IsNDSUNetworkLayerMessage = (data & 0b10000000) != 0
	ctrl.DestinationAddressPresent = (data & 0b00100000) != 0
	ctrl.SourceAddressPresent = (data & 0b00001000) != 0
	ctrl.IsBACNetConfirmedRequestPDUPresent = (data & 0b00000100) != 0
	ctrl.Priority = (NetworkMessagePriority)(data & 0b00000011)

	return ctrl
}

func writeAddress(buf *bytes.Buffer, addr *Address) error {
	if e := writeDoubleByte(buf, addr.Network); e != nil {
		return e
	}
	if e := buf.WriteByte(addr.Length); e != nil {
		return e
	}
	if addr.Length > 0 {
		// Do we need this? Otherwise, maybe we pass in nil to ByteBuffer.Write?
		if _, e := buf.Write(addr.Addr); e != nil {
			return e
		}
	}
	return nil
}

func readAddress(buf *bytes.Buffer) (*Address, error) {
	var addr Address
	db, e := readDoubleByte(buf)
	if e != nil {
		return nil, e
	}
	addr.Network = db
	b, e := buf.ReadByte()
	if e != nil {
		return nil, e
	}
	addr.Length = b
	if addr.Length > 0 {
		// read uses len, not capacity
		addrBuf := make([]byte, addr.Length)
		bytesRead, e := buf.Read(addrBuf)
		if e != nil {
			return nil, e
		}
		if bytesRead != int(addr.Length) {
			return nil, fmt.Errorf("read %d bytes, expected %d bytes", bytesRead, addr.Length)
		}
		addr.Addr = addrBuf
	}
	return &addr, nil
}

// Encode a message
func (m *Message) Encode() ([]byte, error) {
	// We only know that it will be 2 bytes plus data.
	b := make([]byte, 0, 3)
	buf := bytes.NewBuffer(b)
	if e := buf.WriteByte(m.ProtocolVersion); e != nil {
		return nil, e
	}
	if e := buf.WriteByte(encodeControl(m.Control)); e != nil {
		return nil, e
	}

	if m.Control.DestinationAddressPresent {
		if e := writeAddress(buf, m.Destination); e != nil {
			return nil, e
		}
	}
	if m.Control.SourceAddressPresent {
		if e := writeAddress(buf, m.Source); e != nil {
			return nil, e
		}
	}
	if m.Control.DestinationAddressPresent {
		if e := buf.WriteByte(*m.HopCount); e != nil {
			return nil, e
		}
	}
	// vendorID goes in between message type and APDU, but I don't which one it accompanies.
	if m.Control.IsNDSUNetworkLayerMessage {
		if e := buf.WriteByte((byte)(m.MessageType)); e != nil {
			return nil, e
		}
	} else if m.APDU != nil {
		apduBytes, e := m.APDU.Encode()
		if e != nil {
			return nil, e
		}
		if _, e = buf.Write(apduBytes); e != nil {
			return nil, e
		}
	} else {
		return nil, fmt.Errorf("Message was not a NDSU, but does not have application data (APDU)")
	}

	return buf.Bytes(), nil
}
