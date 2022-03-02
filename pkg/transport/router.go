package transport

import (
	"context"
	"errors"
	"sync"

	"github.com/shigmas/modore/internal/apdu"
	"github.com/shigmas/modore/internal/npdu"
)

type (
	// MessageNexus handles message routing and will accept register requests.
	MessageNexus struct {
		bvlcRegistry map[uint8][]BVLCMessageHandler
		bvlcMux      sync.RWMutex
		npduRegistry map[uint8][]NPDUMessageHandler
		npduMux      sync.RWMutex
		apduRegistry map[uint8][]APDUMessageHandler
		apduMux      sync.RWMutex

		wg             sync.WaitGroup
		stopFunc       context.CancelFunc
		defaultHandler *BVLCNPDURouterHandler
	}

	// BVLCNPDURouterHandler handles registers itself with the MessageNexus to handle BVLCMessages and NPDU
	// messages. It will use the nexus's registry to check for other handlers as well. (it will find itself
	// in the registry, although it doesn't really matter.
	BVLCNPDURouterHandler struct {
		registrar MessageRegistrar
		bvlcCh    BVLCMessageChannel
		npduCh    NPDUMessageChannel
	}
)

var (
	_ MessageRouter      = (*MessageNexus)(nil)
	_ MessageRegistrar   = (*MessageNexus)(nil)
	_ BVLCMessageHandler = (*BVLCNPDURouterHandler)(nil)
)

func newBVLCNPDURouterHandler(reg MessageRegistrar) *BVLCNPDURouterHandler {
	return &BVLCNPDURouterHandler{
		registrar: reg,
		bvlcCh:    make(BVLCMessageChannel, 1),
		npduCh:    make(NPDUMessageChannel, 1),
	}
}
func (b *BVLCNPDURouterHandler) GetBVLCChannel() BVLCMessageChannel {
	return b.bvlcCh
}

func (b *BVLCNPDURouterHandler) GetNPDUChannel() NPDUMessageChannel {
	return b.npduCh
}

func (b *BVLCNPDURouterHandler) Equals(other Equatable) bool {
	if o, ok := other.(*BVLCNPDURouterHandler); ok {
		return b == o
	}
	return false
}

func (b *BVLCNPDURouterHandler) getNPDUMessageFromBVLCMessage(msg *BVLCMessage) (npdu.Message, error) {
	// I think only broadcast and unicast messages can have NPDU? Forward also does, but we
	// don't forward.
	if msg.Function != BVLCFunctioncBroadcast && msg.Function != BVLCFunctioncUnicast {
		return nil, errors.New("Invalid BVLCFunction type for this handler")
	}
	return npdu.NewMessageFromBytes(msg.Data)
}

func (b *BVLCNPDURouterHandler) Start(done <-chan struct{}, wg *sync.WaitGroup) {
	// Need to put some add and waits. (not all of them here, though)
	go func() {
		for {
			select {
			case bvlcMsg := <-b.bvlcCh:
				npduMsg, err := b.getNPDUMessageFromBVLCMessage(bvlcMsg)
				if err != nil {
					// implement some error handling for this
					continue
				}
				//for filter, npduHandlers := range b.registrar.GetNPDUHandlers() {
				for _, npduHandlers := range b.registrar.GetNPDUHandlers() {
					// Damn it. type can be 0, which can't be &'ed
					//					if filter&uint8(npduMsg.GetMessageType()) > 0 {
					for _, h := range npduHandlers {
						h.GetNPDUChannel() <- npduMsg
					}
					//				}
				}
			case <-done:
				wg.Done()
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case npduMsg := <-b.npduCh:
				apduMsg := npduMsg.GetAPDUMessage()
				unconfirmed, ok := apduMsg.(*apdu.UnconfirmedMessage)
				if !ok {
					continue
				}
				for filter, apduHandlers := range b.registrar.GetAPDUHandlers() {
					if filter&uint8(unconfirmed.ServiceID) > 0 {
						for _, h := range apduHandlers {
							h.GetAPDUChannel() <- &apduMsg
						}
					}
				}

			case <-done:
				wg.Done()
				return
			}
		}
	}()
}

func NewMessageNexus() *MessageNexus {
	nexus := MessageNexus{
		bvlcRegistry: make(map[uint8][]BVLCMessageHandler),
		npduRegistry: make(map[uint8][]NPDUMessageHandler),
		apduRegistry: make(map[uint8][]APDUMessageHandler),
	}
	nexus.defaultHandler = newBVLCNPDURouterHandler(&nexus)
	nexus.RegisterBVLCHandler(BVLCFunctioncBroadcast|BVLCFunctioncUnicast, nexus.defaultHandler)
	nexus.RegisterNPDUHandler(npdu.NetworkLayerWhoIsMessage|npdu.NetworkLayerIAmMessage, nexus.defaultHandler)

	return &nexus
}

func (n *MessageNexus) Start() {
	ctx, stopFunc := context.WithCancel(context.Background())
	n.wg.Add(2)
	n.defaultHandler.Start(ctx.Done(), &n.wg)
	n.stopFunc = stopFunc
}

func (n *MessageNexus) Stop() {
	n.stopFunc()
	n.wg.Wait()
}

func (n *MessageNexus) RouteMessage(message *BVLCMessage) error {
	n.bvlcMux.RLock()
	defer n.bvlcMux.RUnlock()

	for filter, handlers := range n.bvlcRegistry {
		if filter&uint8(message.Function) != 0 {
			// filter match. Iterate through the handlers and pass the message
			for _, handler := range handlers {
				handler.GetBVLCChannel() <- message
			}
		}
	}
	return nil
}

func (n *MessageNexus) RegisterBVLCHandler(newFilter BVLCFunction, handler BVLCMessageHandler) {
	registerGeneric(uint8(newFilter), handler, n.bvlcRegistry, &n.bvlcMux)
}

func (n *MessageNexus) RegisterNPDUHandler(newFilter npdu.NetworkLayerMessageType, handler NPDUMessageHandler) {
	registerGeneric(uint8(newFilter), handler, n.npduRegistry, &n.npduMux)
}

func (n *MessageNexus) RegisterAPDUHandler(newFilter apdu.ServiceUnconfirmed, handler APDUMessageHandler) {
	registerGeneric(uint8(newFilter), handler, n.apduRegistry, &n.apduMux)
}

func (n *MessageNexus) GetBVLCHandlers() map[uint8][]BVLCMessageHandler {
	return n.bvlcRegistry
}

func (n *MessageNexus) GetNPDUHandlers() map[uint8][]NPDUMessageHandler {
	return n.npduRegistry
}
func (n *MessageNexus) GetAPDUHandlers() map[uint8][]APDUMessageHandler {
	return n.apduRegistry
}

// This is not a great use of generics. But, since we are inserting into a collection (or, even, a
// collection of a collection), generics lets us keep the type of the collection while still using only one
// function instead of 3.
// Also, hide this so the API hides 1.18'isms.
func registerGeneric[HandlerType Equatable](newFilter uint8, handler HandlerType,
	handlerMap map[uint8][]HandlerType, mux *sync.RWMutex) {
	mux.RLock()
	var writeHandlers []HandlerType
	for filter, handlers := range handlerMap {
		if filter == uint8(newFilter) {
			writeHandlers = handlers
			break
		}
	}
	mux.RUnlock()

	mux.Lock()
	defer mux.Unlock()
	if writeHandlers != nil {
		if !isRegistered(handler, writeHandlers) {
			writeHandlers = append(writeHandlers, handler)
		}
	} else {
		writeHandlers = []HandlerType{handler}
	}
	handlerMap[uint8(newFilter)] = writeHandlers
}

func isRegistered[HandlerType Equatable](handler HandlerType, existing []HandlerType) bool {

	for _, h := range existing {
		if h.Equals(handler) {
			return true
		}
	}
	return false
}
