package udphp

import (
	"encoding/json"
	"errors"
	"net"
)

type UDPMessage struct {
	Path    string            `json:"path"`
	PeerID  string            `json:"peerID"`
	Error   string            `json:"error"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
	Addr    *net.UDPAddr
}

func (r *UDPMessage) GetPath() string {
	return r.Path
}

func (r *UDPMessage) GetPeerID() string {
	return r.PeerID
}

func (r *UDPMessage) GetError() error {
	if r.Error == "" {
		return nil
	}

	return errors.New(r.Error)
}

func (r *UDPMessage) GetHeader(s string) (string, error) {
	h, ok := r.Headers[s]
	if !ok {
		return "", ErrHeaderNotFound
	}

	return h, nil
}

func (r *UDPMessage) RawBody() []byte {
	return r.Body
}

func (r *UDPMessage) GetAddr() net.Addr {
	return r.Addr
}
func (r *UDPMessage) SetAddr(addr net.Addr) {
	r.Addr = addr.(*net.UDPAddr)
}

func (r *UDPMessage) Bytes() []byte {
	b, _ := json.Marshal(r)
	return b
}

func NewUDPMessage(request []byte) (Message, error) {
	var udpMessage UDPMessage
	if err := json.Unmarshal(request, &udpMessage); err != nil {
		return nil, err
	}

	return &udpMessage, nil
}

func NewUDPErrMessage(err error) Message {
	return &UDPMessage{Error: err.Error()}
}
