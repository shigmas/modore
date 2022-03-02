package transport

import (
	"github.com/shigmas/modore/internal/apdu"
	"github.com/shigmas/modore/internal/npdu"
)

type (
	// Implementing and registering this interface will receive all messages of this type. Registering
	// can be done for a specific message type or types. Most messages will be filtered sequentially by
	// type.
	// Because of the nature of UDP, bothf requests from clients (or peers) and responses from our
	// requests will both come through this type of system. If we need a specific resonse, the client
	// will have to filter the messages that they are looking for.
	//
	// Since we may get messages that we aren't interested in, we can just disgard them. in that
	// case, we have a ephemeral handler that can timeout, with an optional type.

	// ErrorHandler is implemented by handlers that wish to handle errors. Any handler may implement
	// this. They will be called for all errors at a particular level. This is more useful for epheramal
	// handlers.
	ErrorHandler interface {
		GetErrorChannel() chan error
	}

	Equatable interface {
		Equals(other Equatable) bool
	}

	// BVLCMessageChannel
	BVLCMessageChannel chan *BVLCMessage
	// BVLCMessageHandler can handle very low level messages.
	BVLCMessageHandler interface {
		Equatable
		GetBVLCChannel() BVLCMessageChannel
	}

	NPDUMessageChannel chan npdu.Message
	// NPDUMessageHandler can filter on network level messages
	NPDUMessageHandler interface {
		Equatable
		GetNPDUChannel() NPDUMessageChannel
	}

	APDUMessageChannel chan *apdu.Message
	// APDUMessageHandler will probably be what users typically use. Typically filtered on the
	// type of message that you want to handle. Applications will filter on certain types. WhoIs
	// is a required message to handle. For convenience, a base struct which subscribes to the
	// required BVLC and NPDU messages is provided.
	APDUMessageHandler interface {
		Equatable
		GetAPDUChannel() APDUMessageChannel
	}

	// NPDUMessageHandlerBase handles the correct BVLCMessages that get hold NPDU Messages.
	NPDUMessageHandlerBase struct {
		NPDUMessageHandler
		connection  Connection
		bvlcChannel BVLCMessageChannel
	}

	// APDUMessageHandlerBase handles NPDU message types that have APDU messages. Thus, it uses
	// the NPDUMessageHandlerBase to handle the proper BVLCMessages that have NPDU messages.
	APDUMessageHandlerBase struct {
		APDUMessageHandler
		*NPDUMessageHandlerBase
		npduChannel NPDUMessageChannel
	}
)

var (
	_ BVLCMessageHandler = (*NPDUMessageHandlerBase)(nil)
	_ BVLCMessageHandler = (*APDUMessageHandlerBase)(nil)
)

func NewNPDUMessageHandlerBase(c Connection) *NPDUMessageHandlerBase {
	handler := NPDUMessageHandlerBase{
		connection:  c,
		bvlcChannel: make(BVLCMessageChannel, 1),
	}
	//c.RegisterBVLCHandler(BVLCFunctioncUnicast|BVLCFunctioncBroadcast, &handler)
	return &handler
}

func (h *NPDUMessageHandlerBase) GetBVLCChannel() BVLCMessageChannel {
	return h.bvlcChannel
}

func (h *APDUMessageHandlerBase) NewAPDUMessageHandlerBase(c Connection) *APDUMessageHandlerBase {
	handler := APDUMessageHandlerBase{
		NPDUMessageHandlerBase: NewNPDUMessageHandlerBase(c),
		npduChannel:            make(NPDUMessageChannel, 1),
	}
	//c.RegisterBVLCHandler(transport.BVLCFunctioncUnicast|transport.BVLCFunctioncBroadcast, &handler)
	return &handler
}

func (h *APDUMessageHandlerBase) GetNPDUChannel() NPDUMessageChannel {
	return h.npduChannel
}
