package protocol

import (
	"fmt"

	"io"
	"sync"
)

type Protocol interface {
	Config(string) error
	NewCodec( io.ReadWriter) (Codec, error)
}

type Codec interface {
	Receive() (interface{}, error)
	Send(interface{}) error
	Close() error
}

var (
	procotolMux sync.RWMutex
	adapters = make(map[string]Protocol)
)

func Register(name string, adpater Protocol) {
	procotolMux.Lock()
	defer procotolMux.Unlock()
	if adpater == nil {
		panic("Protocol:adapter is nil")
	}

	if _, ok := adapters[name]; ok {
		panic("Protocol:Register called twice for adapter" + name)
	}
	adapters[name] = adpater
}

func NewProtocol(name string, config string) (Protocol, error) {
	adapter, ok := adapters[name]
	if !ok {
		err := fmt.Errorf("Protocol: unknown adapter name %q (forgot to import?)", name)
		return nil, err
	}

	err := adapter.Config(config)
	if err != nil {
		return nil, err
	}
	return adapter, nil
}
