package tcp

import "net"

type Tcp struct {
	maxConn      int
	listener     net.Listener
	sendChanSize int

	protocol protocol.Protocol

}
