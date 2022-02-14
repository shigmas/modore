package apdu

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
	// APDUMessages can be (but are not always) the data portion of NPDU (Network) data.
	APDUMessage struct {
		// I think this can just be service.
		UnconfirmedService ServiceUnconfirmed
		Data               []byte // raw data
	}
)
