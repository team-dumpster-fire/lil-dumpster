package state

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

type Memory map[string][]byte

var memoryMutext = sync.Mutex{}

func NewMemory() *Memory {
	return &Memory{}
}

func (s Memory) Set(ctx context.Context, key string, value interface{}) (err error) {
	memoryMutext.Lock()
	defer memoryMutext.Unlock()

	s[key], err = json.Marshal(value)
	return
}

func (s Memory) Get(ctx context.Context, key string, value interface{}) error {
	if data, ok := s[key]; !ok {
		return errors.New("key not found")
	} else {
		return json.Unmarshal(data, value)
	}
}
