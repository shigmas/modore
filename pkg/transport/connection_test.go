package transport

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shigmas/modore/internal/apdu"
	"github.com/shigmas/modore/internal/npdu"
)

type (
	TestHandler struct {
		apduChan APDUMessageChannel
		msg      *apdu.Message
	}

	TestRouter struct {
		listenCh chan *BVLCMessage
	}
)

var _ MessageRouter = (*TestRouter)(nil)
var _ APDUMessageHandler = (*TestHandler)(nil)

// Very crappy implementation. But, this was easy to adapt to the old test.
func NewTestRouter(listenCh chan *BVLCMessage) *TestRouter {
	return &TestRouter{listenCh}
}

func (r *TestRouter) Start(doneCh <-chan struct{}) error {
	for {
		select {
		case <-r.listenCh:
			return nil
		case <-doneCh:
			return fmt.Errorf("No response")
		}
	}
}

func (r *TestRouter) RouteMessage(message *BVLCMessage) error {
	r.listenCh <- message
	return nil
}

func NewTestHandler(ch APDUMessageChannel) *TestHandler {
	return &TestHandler{ch, nil}
}

func (h *TestHandler) GetAPDUChannel() APDUMessageChannel {
	return h.apduChan
}

func (h *TestHandler) Equals(other Equatable) bool {
	if o, ok := other.(*TestHandler); ok {
		return h == o
	}
	return false
}

func (h *TestHandler) Start(doneCh <-chan struct{}, wg *sync.WaitGroup) {
	for {
		select {
		case msg := <-h.apduChan:
			h.msg = msg
		case <-doneCh:
			wg.Done()
			return
		}
	}

}

func TestNewConnection(t *testing.T) {
	addr := []byte{192, 168, 3, 16}
	conn, err := NewConnection(addr, 24)
	assert.NoError(t, err, "Unexpected error creating connection")
	realConn, ok := conn.(*connection)
	assert.True(t, ok, "Unable to cast to concrete type")
	assert.NotNil(t, realConn.bacnetConn, "Unexpected nil for the UDP connection")
	assert.Equal(t, realConn.broadcastIP[0], addr[0], "Addr and broadcast don't match")
	assert.Equal(t, realConn.broadcastIP[1], addr[1], "Addr and broadcast don't match")
	assert.Equal(t, realConn.broadcastIP[2], addr[2], "Addr and broadcast don't match")
	assert.Equal(t, realConn.broadcastIP[3], uint8(0xFF), "Addr and broadcast don't match")
	assert.NoError(t, conn.Close(), "Error closing connection")
}

func TestStartStopConnection(t *testing.T) {
	addr := []byte{192, 168, 3, 16}
	conn, err := NewConnection(addr, 24)
	assert.NoError(t, err, "Unexpected error creating connection")
	t.Run("test no receive", func(t *testing.T) {
		r := NewTestRouter(make(chan *BVLCMessage, 1))
		conn.SetMessageRouter(r)
		ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancelFunc()
		assert.Error(t, r.Start(ctx.Done()), "Expected timeout in waiting for data")
	})
	t.Run("TestStartStop", func(t *testing.T) {
		conn.Start()
		conn.Stop()
	})
	assert.NoError(t, conn.Close(), "Error closing connection")
}

// This test doesn't always receive its who is back, so don't run it for CI
func NoTestWhoIs(t *testing.T) {
	addr := []byte{192, 168, 3, 16}
	conn, err := NewConnection(addr, 24)
	assert.NoError(t, err, "Unexpected error in getting connection")

	var wg sync.WaitGroup
	// Create our test handler to catch the message
	ch := make(APDUMessageChannel)
	apduHandler := NewTestHandler(ch)
	//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	wg.Add(1)
	go apduHandler.Start(ctx.Done(), &wg)

	r := NewMessageNexus()
	r.RegisterAPDUHandler(apdu.ServiceUnconfirmedIAm|apdu.ServiceUnconfirmedWhoIs, apduHandler)
	r.Start()
	defer r.Stop()
	conn.SetMessageRouter(r)

	// I think this can be moved to a function to hide APDU. So, maybe some more stuff gets hidden.
	appMsg, err := apdu.NewWhoisMessage(0, 999)
	assert.NoError(t, err, "Unexpected error creating WhoIs Message")
	conn.Start()
	defer func() {
		conn.Stop()
		conn.Close()
	}()
	assert.NoError(t, conn.SendUnconfirmedMessage(nil, npdu.NormalMessage, npdu.NetworkLayerWhoIsMessage, appMsg),
		"Unexpected error sending who is")
	wg.Wait()
	assert.NotNil(t, apduHandler.msg, "Message Never received")

}
