package tcp

import "net"

type Tcp struct {
	maxConn      int64
	listener     net.Listener
	sendChanSize int

}
