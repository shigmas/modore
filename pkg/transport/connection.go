package transport

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/shigmas/modore/internal/apdu"
	"github.com/shigmas/modore/internal/npdu"
)

// Just some experimentation
// Do we want a client and server package? Maybe there isn't much difference. We have to listen for responses
// and maybe that's the same as listening for requests.
//
// We have to encode our addresses into the NPDU format. Actually, npdu should do that.
//
// So, the network package has nothing to do with encoding. Only sending and receiving bytes.
// Then, the work this does is holding the connections and making them channels.

// This is also were we hold the default values.

const (
	// DefaultPort is the default BACnet port. Get it? BAC...
	DefaultPort = 0xBAC0
	// DefaultHopCount is the max? number of hops?
	DefaultHopCount uint8 = 0xFF

	udpNetwork = "udp4"
)

type (
	// All messages are passed through here. Applications must register handlers
	MessageRouter interface {
		RouteMessage(message *BVLCMessage) error
	}

	// Register to receive messages.
	MessageRegistrar interface {
		RegisterBVLCHandler(filter BVLCFunction, handler BVLCMessageHandler)
		RegisterNPDUHandler(filter npdu.NetworkLayerMessageType, handler NPDUMessageHandler)
		RegisterAPDUHandler(filter apdu.ServiceUnconfirmed, handler APDUMessageHandler)
		GetBVLCHandlers() map[uint8][]BVLCMessageHandler
		GetNPDUHandlers() map[uint8][]NPDUMessageHandler
		GetAPDUHandlers() map[uint8][]APDUMessageHandler
	}

	// Connection is the interface for connection to BACnet
	Connection interface {
		SetMessageRouter(r MessageRouter)
		// If Start is called, Stop must also be called.
		Start()
		Stop()
		// But, we always need to call close
		Close() error
		SourceAddress() *npdu.Address
		BroadcastAddress() *npdu.Address
		DestinationAddress(dest net.IP) *npdu.Address
		// These will change in the future, I think
		SendConfirmedMessage(priority npdu.NetworkMessagePriority, msgType npdu.NetworkLayerMessageType, msg *apdu.ConfirmedMessage) error
		SendUnconfirmedMessage(priority npdu.NetworkMessagePriority, msgType npdu.NetworkLayerMessageType, msg *apdu.UnconfirmedMessage) error
	}

	connection struct {
		wg           sync.WaitGroup
		stopFunction func()
		ip4Addr      net.IP
		mask         uint16
		bacnetConn   *net.UDPConn // BACnet is UDP, so this is "the" connection
		broadcastIP  net.IP
		router       MessageRouter
	}

	incomingData struct {
		err    error
		sender *net.UDPAddr
		data   []byte
	}
)

var _ Connection = (*connection)(nil)

// NewConnection creates a connection that will send and receive from the specified IP/mask. We'll need
// a more flexible way that can take the *type/class* of interface
func NewConnection(ip4Addr []byte, netMask uint16) (Connection, error) {
	mask := net.CIDRMask((int)(netMask), 32)
	ip := net.IP(ip4Addr)

	broadcast := net.IP(make([]byte, 4))
	for i := range ip {
		broadcast[i] = ip[i] | ^mask[i]
	}

	udp, err := net.ResolveUDPAddr(udpNetwork, fmt.Sprintf(":%d", DefaultPort))
	if err != nil {
		return nil, fmt.Errorf("unable to resolve UDP Address for port %d: %w", DefaultPort, err)
	}
	conn, err := net.ListenUDP("udp", udp)
	if err != nil {
		return nil, fmt.Errorf("unable to listen on UDP: %w", err)
	}
	return &connection{
		bacnetConn:  conn,
		broadcastIP: broadcast,
	}, nil
}

func (c *connection) SetMessageRouter(r MessageRouter) {
	c.router = r
}

func (c *connection) Start() {
	ctx, stopFunc := context.WithCancel(context.Background())
	dataChannel := make(chan incomingData, 1)
	go c.startListener(dataChannel)
	c.wg.Add(1)
	go c.loopForever(ctx.Done(), dataChannel)
	c.stopFunction = stopFunc
}

// loop forever.
func (c *connection) startListener(ch chan<- incomingData) {
	for {
		b := make([]byte, 2048)
		// this doesn't block, I guess. So, just loop. If we need to, we can add a pause, I guess.
		i, adr, err := c.bacnetConn.ReadFromUDP(b)
		if i > 0 {
			fmt.Printf("Received %d bytes: %s\n", i, string(b[:i]))
			ch <- incomingData{err, adr, b[:i]}
		}
	}
}

func (c *connection) loopForever(doneCh <-chan struct{}, listenCh <-chan incomingData) {
	defer c.wg.Done()
	for {
		select {
		case incoming := <-listenCh:
			if incoming.err != nil {
				fmt.Println("Received error: ", incoming.err)
			} else {
				msg, err := NewBVLCMessageFromBytes(incoming.data)
				if err != nil {
					return
				}
				fmt.Printf("msg function: %d\n", msg.Function)
				if err = c.router.RouteMessage(msg); err != nil {
					fmt.Printf("RouteMessage Error: %v\n", err)
				}
			}
		case <-doneCh:
			return
		}
	}

}
func (c *connection) Stop() {
	if c.stopFunction != nil {
		c.stopFunction()
		c.wg.Wait()
	}
}

func (c *connection) Close() error {
	return c.bacnetConn.Close()
}

// SourceAddress converts from an IP to the npdu.Address type to be encoded.
func (c *connection) SourceAddress() *npdu.Address {
	addrBytes := append(c.ip4Addr, apdu.EncodeUint(DefaultPort, 2)...)
	return &npdu.Address{
		Network: 0,
		Length:  net.IPv4len + 2,
		Addr:    addrBytes,
	}
}

func (c *connection) BroadcastAddress() *npdu.Address {
	addrBytes := append(c.broadcastIP, apdu.EncodeUint(DefaultPort, 2)...)
	return &npdu.Address{
		Network: 0,
		Length:  0, // somehow, we don't really need length in these situations. Such is BACnet
		Addr:    addrBytes,
	}
}

func (c *connection) DestinationAddress(dest net.IP) *npdu.Address {
	addrBytes := append(dest, apdu.EncodeUint(DefaultPort, 2)...)
	return &npdu.Address{
		Network: 0,
		Length:  net.IPv4len + 2,
		Addr:    addrBytes,
	}
}

func (c *connection) udpAddr(ipAddr net.IP) net.Addr {
	return &net.UDPAddr{
		IP:   ipAddr,
		Port: DefaultPort,
	}
}

func (c *connection) SendConfirmedMessage(priority npdu.NetworkMessagePriority,
	msgType npdu.NetworkLayerMessageType, msg *apdu.ConfirmedMessage) error {
	return fmt.Errorf("Not implemented")
}

// SendUnconfirmedMessage will be adapted as I hardcode less stuff
// This handles all three "layers": APDU, NPDU, and BVLC. If we continue to do it like this, we
// can have one byte stream that eventually gets sent over the UDP connection.
func (c *connection) SendUnconfirmedMessage(priority npdu.NetworkMessagePriority,
	msgType npdu.NetworkLayerMessageType, msg *apdu.UnconfirmedMessage) error {

	control := npdu.NewControl(priority, false, false, false, false)
	npduMsg := npdu.NewMessage(control, nil, nil, DefaultHopCount, msgType, nil, msg)

	npduBytes, err := npduMsg.Encode()
	if err != nil {
		return err
	}

	// I think this is either unicast or broadcast. But, this should be passed in.
	bvlcMsg := NewBVLCMessage(BVLCFunctioncBroadcast, npduBytes)
	msgBytes := bvlcMsg.Encode()
	bytesWritten, err := c.bacnetConn.WriteTo(msgBytes, c.udpAddr(c.broadcastIP))
	if err != nil {
		return err
	}
	if bytesWritten != len(msgBytes) {
		return fmt.Errorf("NPDU had %d bytes but only %d were written", len(msgBytes), bytesWritten)
	}
	return nil
}
