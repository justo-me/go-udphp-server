package udphp

import (
	"net"
)

type UDPConnection struct {
	send chan Message
	addr *net.UDPAddr
}

func (c *UDPConnection) Send(message Message) error {
	c.send <- message
	return nil
}

func (c *UDPConnection) GetAddr() net.Addr {
	return c.addr
}

func NewUDPConn(send chan Message, addr *net.UDPAddr) Connection {
	return &UDPConnection{
		send: send,
		addr: addr,
	}
}