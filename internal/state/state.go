package state

import "context"

type Backend interface {
	Set(ctx context.Context, key string, value interface{}) (err error)
	Get(ctx context.Context, key string, value interface{}) error
}
