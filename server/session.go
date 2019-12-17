package server

import (
	"sync"

	"github.com/gary163/seals/protocol"
)

type SessionsMap struct {
	sessions map[int64]*Session
	mu sync.RWMutex
}

type Session struct {
	id int64
	sendChan chan interface{}
	protocol protocol.Protocol
	closed bool
	sendMu sync.Mutex
	recvMu sync.Mutex
}