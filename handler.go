package udphp

import "context"

type HandlerFunc func(ctx context.Context, serverConnection Connection, req Message) (Message, error)
