package udphp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type UDPServer struct {
	conn     *net.UDPConn
	exit     chan bool
	wg       *sync.WaitGroup
	send     chan Message
	handlers map[string]HandlerFunc

	connections Connections
}

func (s *UDPServer) Stop() {
	close(s.exit)
	s.wg.Wait()
}

func (s *UDPServer) Listen() {
	go s.sender()

	s.receiver()
}

func (s *UDPServer) Handle(path string, handlerFunc HandlerFunc) {
	s.handlers[path] = handlerFunc
}

func (s *UDPServer) sender() {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		select {
		case <-s.exit:
			return
		case m := <-s.send:
			_, err := s.conn.WriteToUDP(m.Bytes(), m.GetAddr())
			if err != nil {

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
	var m Message
	err := json.Unmarshal(b, &m)
	if err != nil {
		// todo fix
		c.Send(NewUDPErrMessage(err))
		return
	}

	h, ok := s.handlers[m.GetPath()]
	if !ok {
		c.Send(NewUDPErrMessage(errors.New("handler not found")))
		return
	}

	go h(context.Background(), m)
}

func NewUDPServer(port int) (Server, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	return &UDPServer{
		conn:        conn,
		exit:        make(chan bool),
		wg:          &sync.WaitGroup{},
		send:        make(chan Message, 100),
		handlers:    make(map[string]HandlerFunc),
		connections: make(map[string]Connection),
	}, nil
}
