package udphp

type Server interface {
	Stop()
	Listen()
	Handle(path string, handlerFunc HandlerFunc)
}

