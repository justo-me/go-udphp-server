package udphp

import "context"

type HandlerFunc func(ctx context.Context, serverConnection Connection, req Message) (Message, error)

type Handler struct {
	handlers map[string]HandlerFunc
}

func (h *Handler) Handle(path string, handlerFunc HandlerFunc) {
	h.handlers[path] = handlerFunc
}

func (h *Handler) Handlers() map[string]HandlerFunc {
	return h.handlers
}

func NewHandler() *Handler {
	return &Handler{
		handlers: make(map[string]HandlerFunc),
	}
}
