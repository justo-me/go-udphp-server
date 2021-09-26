package udphp

import (
	"errors"
	"net"
)

var (
	ErrConnectionIsNil = errors.New("connection is nil")
)

type Server interface {
	Stop()
	Listen()
	CreateConnection(addr net.Addr) (Connection, error)
	Handle(path string, handlerFunc HandlerFunc)
}

