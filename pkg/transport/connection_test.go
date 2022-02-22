package transport

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shigmas/modore/internal/apdu"
	"github.com/shigmas/modore/internal/npdu"
)

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
		recvCh := make(chan struct{}, 1)
		conn.AddReceiveChannel(recvCh)
		ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancelFunc()
		assert.Error(t, waitForResponse(ctx.Done(), recvCh), "Expected timeout in waiting for data")
	})
	t.Run("TestStartStop", func(t *testing.T) {
		conn.Start()
		conn.Stop()
	})
	assert.NoError(t, conn.Close(), "Error closing connection")
}

func waitForResponse(doneCh <-chan struct{}, listenCh <-chan struct{}) error {
	for {
		select {
		case <-listenCh:
			return nil
		case <-doneCh:
			return fmt.Errorf("No response")
		}
	}

}

// Just for now
func TestWhoIs(t *testing.T) {
	addr := []byte{192, 168, 3, 16}
	conn, err := NewConnection(addr, 24)
	assert.NoError(t, err, "Unexpected error in getting connection")

	recvCh := make(chan struct{}, 1)

	// I think this can be moved to a function to hide APDU. So, maybe some more stuff gets hidden.
	appMsg := apdu.NewWhoisMessage(0, 999)
	conn.Start()
	defer func() {
		conn.Stop()
		conn.Close()
	}()
	conn.AddReceiveChannel(recvCh)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFunc()
	assert.NoError(t, conn.SendUnconfirmedMessage(npdu.NormalMessage, npdu.NetworkLayerWhoIsMessage, appMsg),
		"Unexpected error sending who is")
	assert.NoError(t, waitForResponse(ctx.Done(), recvCh), "timeout waiting for whois response")
}
