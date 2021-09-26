package udphp

import (
	"encoding/base64"
	"errors"
	"net"
)

type UDPConnection struct {
	send   chan Message
	addr   *net.UDPAddr
	secret string
}

func (c *UDPConnection) Send(message Message) error {
	message.SetAddr(c.addr)
	c.send <- message
	return nil
}

func (c *UDPConnection) GetAddr() net.Addr {
	return c.addr
}

func (c *UDPConnection) GetSecret() ([]byte, error) {
	return convertSecret(c.secret)
}

func (c *UDPConnection) SetSecret(secret []byte) {
	c.secret = base64.StdEncoding.EncodeToString(secret[:])
}

func convertSecret(secretText string) ([]byte, error) {
	// ensure secret has been set
	var secret []byte
	if secretText == "" {
		return secret, errors.New("secret has not been set")
	}

	// decode to byte slice
	bs, err := base64.StdEncoding.DecodeString(secretText)
	if err != nil {
		return secret, errors.New("could not decode secret")
	}

	// copy byte slice into byte array
	copy(secret[:], bs)
	return secret, nil
}

func NewUDPConn(send chan Message, addr *net.UDPAddr) Connection {
	return &UDPConnection{
		send: send,
		addr: addr,
	}
}
