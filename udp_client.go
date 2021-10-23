package udphp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	ErrPeerConnectionTimeout = errors.New("peer connection timeout")
)

type UDPClient struct {
	self *Peer

	peer sync.Map

	keySent     sync.Map
	keyReceived sync.Map
	s           Server
	sAddr       *net.UDPAddr
	sConn       Connection

	pConn sync.Map

	registeredCallback func(Client)
	connectingCallback func(Client)
	connectedCallback  func(Client)
}

func (c *UDPClient) Handle(handler *Handler) {
	c.s.Handle(handler)
}

func (c *UDPClient) WasKeySent(id string) bool {
	b, _ := c.keySent.LoadOrStore(id, false)
	return b.(bool)
}

func (c *UDPClient) SetKeySent(id string, b bool) {
	c.keySent.Store(id, b)
}

func (c *UDPClient) WasKeyReceived(id string) bool {
	b, _ := c.keyReceived.LoadOrStore(id, false)
	return b.(bool)
}

func (c *UDPClient) SetKeyReceived(id string, b bool) {
	c.keyReceived.Store(id, b)
}

func (c *UDPClient) GetServer() Server {
	return c.s
}

func (c *UDPClient) GetSelf() *Peer {
	return c.self
}

func (c *UDPClient) GetPeer(id string) (*Peer, error) {
	p, ok := c.peer.Load(id)
	if !ok {
		return nil, ErrPeerNotFound
	}

	return p.(*Peer), nil
}

func (c *UDPClient) SetPeer(peer *Peer) {
	c.peer.Store(peer.ID, peer)
}

func (c *UDPClient) GetPeerConn(id string) (Connection, error) {
	conn, ok := c.pConn.Load(id)
	if !ok {
		return nil, ErrPeerConnectionNotFound
	}

	return conn.(Connection), nil
}

func (c *UDPClient) SetPeerConn(id string, connection Connection) {
	c.pConn.Store(id, connection)
}

func (c *UDPClient) GetServerConn() Connection {
	return c.sConn
}

func (c *UDPClient) SetServerConn(connection Connection) {
	c.sConn = connection
}

func (c *UDPClient) Connect(id string) error {

	c.ConnectingCallback()

	self := c.GetSelf()
	pConn, err := c.GetPeerConn(id)
	if err != nil {
		return err
	}

	err = pConn.Send(&UDPMessage{
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
			if c.WasKeyReceived(id) {
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

	return &UDPMessage{
		Path:   RouteRegister,
		PeerID: self.ID,
		Body:   []byte(base64.StdEncoding.EncodeToString(sPubKey[:])),
	}, nil
}

func (c *UDPClient) connectHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {

	self := c.GetSelf()

	pConn, err := c.GetPeerConn(req.GetPeerID())
	if err != nil {
		return nil, err
	}

	if pConn != serverConn {
		if pConn.GetAddr().String() == serverConn.GetAddr().String() {
			pConn = serverConn
		} else {
			return nil, errors.New("received connect message from unknown peer")
		}
	}

	pubKey := self.PublicKey

	c.SetKeySent(req.GetPeerID(), true)

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

	pConn, err := c.GetPeerConn(req.GetPeerID())
	if err != nil {
		return nil, err
	}

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

	c.SetKeyReceived(req.GetPeerID(), true)

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

	c.SetPeerConn(p.ID, pCon)

	err = c.Connect(p.ID)
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
		s:           s,
		self:        self,
		peer:        sync.Map{},
		pConn:       sync.Map{},
		sAddr:       sAddr,
		keyReceived: sync.Map{},
		keySent:     sync.Map{},

		connectedCallback:  func(client Client) {},
		connectingCallback: func(client Client) {},
		registeredCallback: func(client Client) {},
	}

	h := NewHandler()

	h.Handle(RouteGreeting, c.greetingHandler)
	h.Handle(RouteRegister, c.registerHandler)
	h.Handle(RouteEstablish, c.establishHandler)
	h.Handle(RouteConnect, c.connectHandler)
	h.Handle(RouteKey, c.keyHandler)

	c.Handle(h)

	return c, nil
}
