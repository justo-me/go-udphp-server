package udphp

import "net"

type Connection interface {
	Send(message Message) error
	GetAddr() net.Addr
	GetSecret() ([]byte, error)
	SetSecret([]byte)
}

type Connections map[string]Connection