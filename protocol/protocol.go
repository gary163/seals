package protocol

import (
	"fmt"
	"io"
	"sync"
)

type protocol interface {
	Init(rw io.ReadWriter,config string) error
	Receive() (interface{}, error)
	Send(interface{}) error
	Close() error
}

var (
	procotolMux sync.RWMutex
	adapters map[string]protocol
)

func Register(name string, adpater protocol) {
	procotolMux.Lock()
	defer procotolMux.Unlock()
	if adpater == nil {
		panic("Protocol:adapter is nil")
	}

	if _,ok := adapters[name]; ok {
		panic("Protocol:Register called twice for adapter" +name)
	}
	adapters[name] = adpater
}

func newProtocol(name string, config string,rw io.ReadWriter) (protocol,error) {
	adapter, ok := adapters[name]
	if !ok {
		err := fmt.Errorf("Protocol: unknown adapter name %q (forgot to import?)", name)
		return nil,err
	}

	err := adapter.Init(rw, config)
	if err != nil {
		return nil,err
	}
	return adapter,nil
}
