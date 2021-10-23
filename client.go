package udphp

type Client interface {
	WasKeySent(string) bool
	SetKeySent(string, bool)
	WasKeyReceived(string) bool
	SetKeyReceived(string, bool)
	GetServer() Server
	GetSelf() *Peer
	GetPeer(id string) (*Peer, error)
	SetPeer(*Peer)
	GetPeerConn(id string) (Connection, error)
	SetPeerConn(string, Connection)
	GetServerConn() Connection
	SetServerConn(Connection)
	Connect(id string) error
	Stop()
	Start() error
	RegisteredCallback()
	ConnectingCallback()
	ConnectedCallback()
	OnRegistered(func(Client))
	OnConnecting(func(Client))
	OnConnected(func(Client))
	Handle(handler *Handler)
}
