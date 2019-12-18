package tcp

import (
	"net"
	"sync"

	"github.com/gary163/seals/protocol"
	"github.com/gary163/seals/server"
)

const (
	defaultMaxConn = 200000
	defaultSendChanSize = 1024
)

type Tcp struct {
	maxConn      int
	listener     net.Listener
	sendChanSize int
	wg           sync.WaitGroup
	protocol     protocol.Protocol

}

type handle func(server.Session)

