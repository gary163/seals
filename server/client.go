package server

import (
	"fmt"
	"sync"

	"github.com/gary163/seals/protocol"
)

type Client interface {
	Init(string, protocol.Protocol, Handler, *SessionManager) error
	Run()
	Close() error
}

var (
	clientAdpatersMu sync.RWMutex
	clientAdapters = make(map[string]Client)
)


func RegisterClient(name string, adapter Client) {
	clientAdpatersMu.Lock()
	defer clientAdpatersMu.Unlock()
	if adapter == nil {
		panic("Client:adapter is nil")
	}

	if _,ok := adapters[name]; ok {
		panic("Client:Register called twice for adapter" +name)
	}
	clientAdapters[name] = adapter
}

func NewClient(name string, config string, protocol protocol.Protocol, handler Handler) (Client, error){
	adapter, ok := clientAdapters[name]
	if !ok {
		err := fmt.Errorf("Client: unknown adapter name %q (forgot to import?)", name)
		return nil,err
	}
	sm := NewSessionManager()
	err := adapter.Init(config, protocol, handler, sm)
	if err != nil {
		return nil,err
	}

	return adapter,nil
}