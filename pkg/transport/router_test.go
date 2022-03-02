package transport

import (
	"context"
	"testing"
	"time"

	"github.com/shigmas/modore/internal/apdu"
	"github.com/shigmas/modore/internal/npdu"

	"github.com/stretchr/testify/assert"
)

// We don't have a base handler for this type
type (
	testBVLCMessageHandler struct {
		ch BVLCMessageChannel
	}
	testNPDUMessageHandler struct {
		ch NPDUMessageChannel
	}
	testAPDUMessageHandler struct {
		ch APDUMessageChannel
	}
)

var (
	_ (BVLCMessageHandler) = (*testBVLCMessageHandler)(nil)
	_ (NPDUMessageHandler) = (*testNPDUMessageHandler)(nil)
	_ (APDUMessageHandler) = (*testAPDUMessageHandler)(nil)
)

func newTestBVLCMessageHandler() *testBVLCMessageHandler {
	return &testBVLCMessageHandler{
		ch: make(BVLCMessageChannel),
	}
}

func (b *testBVLCMessageHandler) GetBVLCChannel() BVLCMessageChannel {
	return b.ch
}

func (b *testBVLCMessageHandler) Equals(other Equatable) bool {
	if o, ok := other.(*testBVLCMessageHandler); ok {
		return b == o
	}
	return false
}

func newTestNPDUMessageHandler() *testNPDUMessageHandler {
	return &testNPDUMessageHandler{
		ch: make(NPDUMessageChannel),
	}
}

func (n *testNPDUMessageHandler) GetNPDUChannel() NPDUMessageChannel {
	return n.ch
}

func (n *testNPDUMessageHandler) Equals(other Equatable) bool {
	if o, ok := other.(*testNPDUMessageHandler); ok {
		return n == o
	}
	return false
}

func newTestAPDUMessageHandler() *testAPDUMessageHandler {
	return &testAPDUMessageHandler{
		ch: make(APDUMessageChannel),
	}
}

func (a *testAPDUMessageHandler) GetAPDUChannel() APDUMessageChannel {
	return a.ch
}

func (a *testAPDUMessageHandler) Equals(other Equatable) bool {
	if o, ok := other.(*testAPDUMessageHandler); ok {
		return a == o
	}
	return false
}

func TestMessageNexus(t *testing.T) {

	t.Run("TestRegister", func(t *testing.T) {
		nexus := NewMessageNexus()
		bHandler := newTestBVLCMessageHandler()
		nHandler := newTestNPDUMessageHandler()
		aHandler := newTestAPDUMessageHandler()

		// We add default handlers for BVLC and NPDU
		assert.Equal(t, 1, len(nexus.bvlcRegistry), "Unexpected number of entries in BVLC Registry")
		assert.Equal(t, 1, len(nexus.npduRegistry), "Unexpected number of entries in NPDU Registry")
		assert.Equal(t, 0, len(nexus.apduRegistry), "Unexpected number of entries in APDU Registry")

		nexus.RegisterBVLCHandler(BVLCFunctionResult, bHandler)
		assert.Equal(t, 2, len(nexus.bvlcRegistry), "Unexpected number of entries in BVLC Registry")
		nexus.RegisterNPDUHandler(npdu.NetworkLayerIAmMessage, nHandler)
		assert.Equal(t, 1, len(nexus.npduRegistry), "Unexpected number of entries in NDPU Registry")
		nexus.RegisterAPDUHandler(apdu.ServiceUnconfirmedIAm, aHandler)
		assert.Equal(t, 1, len(nexus.apduRegistry), "Unexpected number of entries in APDU Registry")

		// Register for multiple values (and the same handler
		nexus.RegisterBVLCHandler(BVLCFunctionResult|BVLCFunctioncBroadcast, bHandler)
		assert.Equal(t, 2, len(nexus.bvlcRegistry), "Unexpected number of entries in BVLC Registry")
		nexus.RegisterBVLCHandler(BVLCFunctionResult, bHandler)
		assert.Equal(t, 2, len(nexus.bvlcRegistry), "Unexpectedly added the same handler")
	})

	t.Run("TestRoute", func(t *testing.T) {
		nexus := NewMessageNexus()
		bHandler := newTestBVLCMessageHandler()

		msg := NewBVLCMessage(BVLCFunctioncBroadcast, nil)
		nexus.RegisterBVLCHandler(BVLCFunctioncBroadcast, bHandler)
		go nexus.RouteMessage(msg)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		timeout := false
		select {
		case msg := <-bHandler.ch:
			assert.Equal(t, BVLCFunctioncBroadcast, int(msg.Function), "BVLCFunction mismatch")
		case <-ctx.Done():
			timeout = true
		}
		assert.False(t, timeout, "Timeout waiting for message")

	})
}
