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
	ErrAddMustBeUDPAddr = errors.New("could not assert net.Addr to *net.UDPAddr")

	DefaultPeerRepository = NewPeerMemoryRepository
)

const (
	RouteGreeting  = "greeting"
	RouteConnect   = "connect"
	RouteRegister  = "register"
	RouteEstablish = "establish"
	RouteKey       = "key"
	RouteNotFound  = "not-found"
)

type UDPServer struct {
	conn     *net.UDPConn
	exit     chan bool
	wg       *sync.WaitGroup
	send     chan Message
	handlers map[string]HandlerFunc

	peers PeerRepository

	publicKey  []byte
	privateKey []byte

	connections Connections
}

func (s *UDPServer) CreateConnection(addr net.Addr) (Connection, error) {
	if addr == nil {
		return nil, ErrConnectionIsNil
	}

	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return nil, ErrAddMustBeUDPAddr
	}

	c := NewUDPConn(s.send, udpAddr)
	s.connections[addr.String()] = c
	return c, nil
}

func (s *UDPServer) PeerRepository(repository PeerRepository) {
	s.peers = repository
}

func (s *UDPServer) Stop() {
	close(s.exit)
	s.wg.Wait()
}

func (s *UDPServer) Listen() {
	go s.sender()

	s.receiver()
}

func (s *UDPServer) Handle(handler *Handler) {
	handlers := handler.Handlers()
	for path, handlerFunc := range handlers {
		s.handlers[path] = handlerFunc
	}
}

func (s *UDPServer) sender() {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		select {
		case <-s.exit:
			return
		case m := <-s.send:
			_, err := s.conn.WriteToUDP(m.Bytes(), m.GetAddr().(*net.UDPAddr))
			if err != nil {
				// todo add error check
			}
		}
	}
}

func (s *UDPServer) receiver() {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		select {
		case <-s.exit:
			s.conn.Close()
			return
		default:
		}

		buf := make([]byte, 2048)
		err := s.conn.SetDeadline(time.Now().Add(time.Second))
		if err != nil {
			continue
		}

		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}

			delete(s.connections, addr.String())
			return
		}

		c, ok := s.connections[addr.String()]
		if !ok {
			c = NewUDPConn(s.send, addr)
			s.connections[addr.String()] = c
		}

		s.wg.Add(1)
		go s.serve(buf[:n], c)
	}

}

func (s *UDPServer) serve(b []byte, c Connection) {
	defer s.wg.Done()
	var m UDPMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		// todo fix
		c.Send(NewUDPErrMessage(fmt.Errorf("cannot parse incomming message: %w", err)))
		return
	}

	go s.handleMessage(&m, c)
}

func (s *UDPServer) handleMessage(m Message, c Connection) {

	h, ok := s.handlers[m.GetPath()]
	if !ok {
		return
		//c.Send(NewUDPErrMessage(errors.New("handler not found")))
		//return
	}

	res, err := h(context.Background(), c, m)
	if err != nil {
		c.Send(NewUDPErrMessage(err))
		return
	}

	if res != nil {
		c.Send(res)
	}
}

func (s *UDPServer) greetingHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {

	str := string(req.RawBody())

	bs, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}

	var clientPubKey [32]byte
	copy(clientPubKey[:], bs[:])

	conn := serverConn

	secret, err := GenSharedSecret(s.publicKey, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("error generating shared secret: %w", err)
	}

	conn.SetSecret(secret)

	return &UDPMessage{
		Path: RouteGreeting,
		Body: []byte(base64.StdEncoding.EncodeToString(s.publicKey[:])),
	}, nil
}

func (s *UDPServer) registerHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {

	err := s.peers.Create(ctx, &Peer{
		ID:   req.GetPeerID(),
		Addr: serverConn.GetAddr().(*net.UDPAddr),
	})

	if err != nil {
		return nil, err
	}

	return &UDPMessage{
		Path: RouteRegister,
	}, nil
}

func (s *UDPServer) notFoundHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {
	return nil, nil
}

func (s *UDPServer) establishHandler(ctx context.Context, serverConn Connection, req Message) (Message, error) {

	rp, err := s.peers.Get(ctx, req.GetPeerID())
	if err != nil {
		return nil, fmt.Errorf("client is not registered with this server: %w", err)
	}

	id := string(req.RawBody())

	op, err := s.peers.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("client is not registered with this server: %w", err)
	}

	connStr := op.Addr.String()

	conn, ok := s.connections[connStr]
	if !ok {
		return nil, errors.New("could not resolve peer connection")
	}

	err = conn.Send(&UDPMessage{
		Path: RouteEstablish,
		Body: MustJson(rp),
	})
	if err != nil {
		return nil, err
	}

	return &UDPMessage{
		Path: RouteEstablish,
		Body: MustJson(op),
	}, nil
}

func NewUDPServer(udpAddr *net.UDPAddr, publicKey []byte, privateKey []byte) (Server, error) {

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	s := &UDPServer{
		conn:        conn,
		exit:        make(chan bool),
		wg:          &sync.WaitGroup{},
		send:        make(chan Message, 100),
		handlers:    make(map[string]HandlerFunc),
		connections: make(map[string]Connection),
		peers:       DefaultPeerRepository(),
		publicKey:   publicKey,
		privateKey:  privateKey,
	}

	h := NewHandler()

	h.Handle(RouteGreeting, s.greetingHandler)
	h.Handle(RouteRegister, s.registerHandler)
	h.Handle(RouteEstablish, s.establishHandler)
	h.Handle(RouteNotFound, s.notFoundHandler)

	s.Handle(h)

	return s, nil
}
