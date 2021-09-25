package udphp

import "net"

type Connection interface {
	Send(message Message) error
	GetAddr() net.Addr
}

type Connections map[string]Connection