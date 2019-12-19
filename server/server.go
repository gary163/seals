package server

import (
	"fmt"
	"sync"

	"github.com/gary163/seals/protocol"
)

type Server interface {
	Init(string, protocol.Protocol, Handler, *SessionManager) error
	Run() error
	Stop() error
}

var (
	adpatersMu sync.RWMutex
 	adapters = make(map[string]Server)
)

func RegisterServer(name string, adapter Server) {
	adpatersMu.Lock()
	defer adpatersMu.Unlock()
	if adapter == nil {
		panic("Server:adapter is nil")
	}

	if _,ok := adapters[name]; ok {
		panic("Server:Register called twice for adapter" +name)
	}
	adapters[name] = adapter
}

func NewServer(name string, config string, protocol protocol.Protocol, handler Handler) (Server, error){
	adapter, ok := adapters[name]
	if !ok {
		err := fmt.Errorf("Server: unknown adapter name %q (forgot to import?)", name)
		return nil,err
	}
	sm := NewSessionManager()//初始化sessionsMap
	err := adapter.Init(config, protocol, handler, sm)
	if err != nil {
		return nil,err
	}

	return adapter,nil
}