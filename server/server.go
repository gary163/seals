package server

import (
	"fmt"
	"sync"
)

type Server interface {
	Init(config string) error
	Run() error
	Stop() error
}

var (
	adpatersMu sync.RWMutex
 	adapters = make(map[string]Server)
)

type Hander interface {
	Handle(session *Session)
}

type HandlerFunc func(session *Session)

func (f HandlerFunc) Handle(session *Session){
	f(session)
}

func Register(name string, adapter Server) {
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

func NewServer(name string, config string) (Server, error){
	adapter, ok := adapters[name]
	if !ok {
		err := fmt.Errorf("Server: unknown adapter name %q (forgot to import?)", name)
		return nil,err
	}

	err := adapter.Init(config)
	if err != nil {
		return nil,err
	}
	return adapter,nil
}