package udphp

import (
	"context"
	"errors"
	"net"
)

var (
	ErrPeerNotFound           = errors.New("peer not found")
	ErrPeerConnectionNotFound = errors.New("peer connection not found")
)

type PeerRepository interface {
	Create(ctx context.Context, peer *Peer) error
	GetAll(ctx context.Context) ([]*Peer, error)
	Get(ctx context.Context, id string) (*Peer, error)
	Remove(ctx context.Context, id string) error
}

type Peer struct {
	ID         string `json:"id"`
	PublicKey  []byte `json:"public_key"`
	PrivateKey []byte `json:"-"`
	Addr       *net.UDPAddr
}

type PeerMemoryRepository struct {
	peers map[string]*Peer
}

func (r *PeerMemoryRepository) Create(ctx context.Context, peer *Peer) error {

	r.peers[peer.ID] = &Peer{
		ID:   peer.ID,
		Addr: peer.Addr,
	}

	return nil
}

func (r *PeerMemoryRepository) GetAll(ctx context.Context) ([]*Peer, error) {
	peers := make([]*Peer, 0, len(r.peers))
	for _, p := range r.peers {
		peers = append(peers, p)
	}

	return peers, nil
}

func (r *PeerMemoryRepository) Get(ctx context.Context, id string) (*Peer, error) {
	p, ok := r.peers[id]
	if !ok {
		return nil, ErrPeerNotFound
	}

	return p, nil
}

func (r *PeerMemoryRepository) Remove(ctx context.Context, id string) error {
	_, ok := r.peers[id]
	if !ok {
		return ErrPeerNotFound
	}

	delete(r.peers, id)

	return nil
}

func NewPeerMemoryRepository() PeerRepository {
	return &PeerMemoryRepository{peers: make(map[string]*Peer)}
}
