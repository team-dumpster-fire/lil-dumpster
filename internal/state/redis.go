package state

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type Redis struct {
	client *redis.Client
}

var redisMutext = sync.Mutex{}

func NewRedis(opt *redis.Options) *Redis {
	return &Redis{client: redis.NewClient(opt)}
}

func (s Redis) Set(ctx context.Context, key string, value interface{}) (err error) {
	redisMutext.Lock()
	defer redisMutext.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, time.Hour*24*90).Err()
}

func (s Redis) Get(ctx context.Context, key string, value interface{}) error {
	cmd := s.client.Get(ctx, key)
	if cmd.Err() != nil {
		return cmd.Err()
	}

	data, err := cmd.Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, value)
}
