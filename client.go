package udphp

type Client interface {
	WasKeySent() bool
	SetKeySent(bool)
	WasKeyReceived() bool
	SetKeyReceived(bool)
	GetServer() Server
	GetSelf() *Peer
	GetPeer() *Peer
	SetPeer(*Peer)
	GetPeerConn() Connection
	SetPeerConn(Connection)
	GetServerConn() Connection
	SetServerConn(Connection)
	Connect() error
	Stop()
	Start() error
	RegisteredCallback()
	ConnectingCallback()
	ConnectedCallback()
	OnRegistered(func(Client))
	OnConnecting(func(Client))
	OnConnected(func(Client))
	Handle(path string, handlerFunc HandlerFunc)
}
