package udphp

import "context"

type HandlerFunc func(ctx context.Context, req Message) (Message, error)
