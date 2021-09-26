package udphp

import "net"

type Peer struct {
	ID         string `json:"id"`
	PublicKey  []byte `json:"public_key"`
	PrivateKey []byte `json:"-"`
	Addr       *net.UDPAddr
}
