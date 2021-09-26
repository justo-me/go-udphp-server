package udphp

import (
	"errors"
	"net"
)

var (
	ErrHeaderNotFound = errors.New("header not found")
	ErrHandlerNotFound = errors.New("handler not found")
)

type Message interface {
	GetPath() string
	GetPeerID() string
	GetError() error
	GetHeader(string) (string, error)
	RawBody() []byte
	GetAddr() net.Addr
	SetAddr(net.Addr)
	Bytes() []byte
}
