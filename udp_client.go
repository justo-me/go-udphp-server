package udphp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
)

var (
	ErrPeerConnectionTimeout = errors.New("peer connection timeout")
)

type UDPClient struct {
	self        *Peer
	peer        *Peer
	keySent     bool
	keyReceived bool
	s           Server
	sAddr       *net.UDPAddr
	sConn       Connection
	pConn       Connection

	registeredCallback func(Client)
	connectingCallback func(Client)
	connectedCallback  func(Client)
}

func (c *UDPClient) Handle(path string, handlerFunc HandlerFunc) {
	c.s.Handle(path, handlerFunc)
}

func (c *UDPClient) WasKeySent() bool {
	return c.keySent
}

func (c *UDPClient) SetKeySent(b bool) {
	c.keySent = b
}

func (c *UDPClient) WasKeyReceived() bool {
	return c.keyReceived
}

func (c *UDPClient) SetKeyReceived(b bool) {
	c.keyReceived = b
}

func (c *UDPClient) GetServer() Server {
	return c.s
}

func (c *UDPClient) GetSelf() *Peer {
	return c.self
}

func (c *UDPClient) GetPeer() *Peer {
	return c.peer
}

func (c *UDPClient) SetPeer(peer *Peer) {
	c.peer = peer
}

func (c *UDPClient) GetPeerConn() Connection {
	return c.pConn
}

func (c *UDPClient) SetPeerConn(connection Connection) {
	c.pConn = connection
}

func (c *UDPClient) GetServerConn() Connection {
	return c.sConn
}

func (c *UDPClient) SetServerConn(connection Connection) {
	c.sConn = connection
}

func (c *UDPClient) Connect() error {

	c.ConnectingCallback()

	self := c.GetSelf()
	pConn := c.GetPeerConn()

	err := pConn.Send(&UDPMessage{
		Path: RouteConnect,
		Body: []byte(self.ID),
	})
	if err != nil {
		return err
	}

	timeout := time.After(15 * time.Second)
	tick := time.Tick(3 * time.Second)

	for {
		select {
		case <-timeout:
			return ErrPeerConnectionTimeout
		case <-tick:
			if c.WasKeyReceived() {
				c.ConnectedCallback()
				return nil
			}
		}
	}

}

func (c *UDPClient) Stop() {
	c.s.Stop()
}

func (c *UDPClient) Start() error {
	s := c.GetServer()

	sConn, err := s.CreateConnection(c.sAddr)
	if err != nil {
		return err
	}

	c.SetServerConn(sConn)

	err = sConn.Send(&UDPMessage{
		Path: RouteGreeting,
		Body: []byte(base64.StdEncoding.EncodeToString(c.GetSelf().PublicKey[:])),
		Addr: c.sAddr,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *UDPClient) RegisteredCallback() {
	c.registeredCallback(c)
}

func (c *UDPClient) ConnectingCallback() {
	c.connectingCallback(c)
}

func (c *UDPClient) ConnectedCallback() {
	c.connectedCallback(c)
}

func (c *UDPClient) OnRegistered(f func(Client)) {
	c.registeredCallback = f
}

func (c *UDPClient) OnConnecting(f func(Client)) {
	c.connectingCallback = f
}

func (c *UDPClient) OnConnected(f func(Client)) {
	c.connectedCallback = f
}

func (c *UDPClient) greetingHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {


	self := c.GetSelf()
	if req.GetError() != nil {
		return nil, req.GetError()
	}

	s := string(req.RawBody())

	bs, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	var pubKey [32]byte
	copy(pubKey[:], bs)

	sPubKey := c.GetSelf().PublicKey
	sharedSecret, err := GenSharedSecret(self.PrivateKey, pubKey[:])
	if err != nil {
		return nil, fmt.Errorf("error generating shared secret: %w", err)
	}

	serverConn.SetSecret(sharedSecret)

	fmt.Println(sharedSecret)

	return &UDPMessage{
		Path:   RouteRegister,
		PeerID: self.ID,
		Body:   []byte(base64.StdEncoding.EncodeToString(sPubKey[:])),
	}, nil
}

func (c *UDPClient) connectHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {

	self := c.GetSelf()

	pConn := c.GetPeerConn()
	if pConn != serverConn {
		if pConn.GetAddr().String() == serverConn.GetAddr().String() {
			pConn = serverConn
		} else {
			return nil, errors.New("received connect message from unknown peer")
		}
	}

	pubKey := self.PublicKey

	c.SetKeySent(true)

	return &UDPMessage{
		Path:   RouteKey,
		PeerID: self.ID,
		Body:   []byte(base64.StdEncoding.EncodeToString(pubKey[:])),
	}, nil
}

func (c *UDPClient) registerHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {

	return &UDPMessage{
		Path:   RouteEstablish,
		PeerID: c.GetSelf().ID,
		Body:   []byte(c.GetSelf().ID),
	}, nil
}

func (c *UDPClient) keyHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {

	pConn := c.GetPeerConn()

	if pConn != serverConn {
		return nil, errors.New("received key message from unknown peer")
	}

	s := string(req.RawBody())

	bs, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	var pubKey [32]byte
	copy(pubKey[:], bs)

	sharedSecret, err := GenSharedSecret(c.GetSelf().PrivateKey, pubKey[:])
	if err != nil {
		return nil, err
	}
	pConn.SetSecret(sharedSecret)

	c.SetKeyReceived(true)

	return nil, nil
}

func (c *UDPClient) establishHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {

	if req.GetError() != nil {
		return nil, req.GetError()
	}

	var p Peer
	err := json.Unmarshal(req.RawBody(), &p)
	if err != nil {
		return nil, err
	}
	c.SetPeer(&p)

	//if c.GetPeerConn() != nil && c.GetPeerConn().GetAddr().String() != c.GetServerConn().GetAddr().String() {
	//	return nil, nil
	//}

	pCon, err := c.GetServer().CreateConnection(p.Addr)
	if err != nil {
		return nil, err
	}

	c.SetPeerConn(pCon)

	err = c.Connect()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// NewUDPClient create udp client with sAddr relay address
func NewUDPClient(ID string, s Server, sAddr *net.UDPAddr,
	publicKey []byte, privateKey []byte) (Client, error) {

	self := &Peer{}
	self.PrivateKey = privateKey
	self.PublicKey = publicKey
	self.ID = ID

	c := &UDPClient{
		s:     s,
		self:  self,
		peer:  &Peer{},
		sAddr: sAddr,

		connectedCallback:  func(client Client) {},
		connectingCallback: func(client Client) {},
		registeredCallback: func(client Client) {},
	}

	c.Handle(RouteGreeting, c.greetingHandler)
	c.Handle(RouteRegister, c.registerHandler)
	c.Handle(RouteEstablish, c.establishHandler)
	c.Handle(RouteConnect, c.connectHandler)
	c.Handle(RouteKey, c.keyHandler)

	return c, nil
}
