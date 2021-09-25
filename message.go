package udphp

import (
	"errors"
	"net"
)

var (
	ErrHeaderNotFound = errors.New("header not found")
)

type Message interface {
	GetPath() string
	GetPeerID() string
	GetError() error
	GetHeader(string) (string, error)
	RawBody() []byte
	GetAddr() *net.UDPAddr
	Bytes() []byte
}
