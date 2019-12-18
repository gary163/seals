package tcp

import (
	"encoding/json"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gary163/seals/protocol"
	"github.com/gary163/seals/server"
)

const (
	defaultMaxConn = 200000
	defaultSendChanSize = 1024
	maxTryTime = 3
)

type tcp struct {
	addr         string
	maxConn      int//最大连接数
	listener     net.Listener
	sendChanSize int//异步send的buffer个数
	wg           sync.WaitGroup
	protocol     protocol.Protocol
	handler      server.Hander
	stop         func()
}

func (server *tcp) Init(config string, protocol protocol.Protocol, handler server.Hander) error {
	var cfg map[string]string
	err := json.Unmarshal([]byte(config),&cfg)
	if err != nil {
		return err
	}

	if _,ok := cfg["maxConn"]; !ok {
		cfg["maxConn"] = strconv.Itoa(defaultMaxConn)
	}

	if _,ok := cfg["sendChanSize"]; !ok {
		cfg["sendChanSize"] = strconv.Itoa(defaultSendChanSize)
	}

	if server.listener,err = net.Listen("tcp",server.addr); err!= nil {
		return err
	}

	server.maxConn,_      = strconv.Atoi(cfg["maxConn"])
	server.sendChanSize,_ = strconv.Atoi(cfg["sendChanSize"])
	server.protocol = protocol
	server.handler  = handler

	return nil
}

func (server *tcp) Run() error {
	tryTime := 0
	for{
		conn,err := server.listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return io.EOF
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() && tryTime < maxTryTime{
				time.Sleep(5*time.Millisecond)
				tryTime ++
				continue
			}
		}

		server.wg.Add(1)
		go func(){

			server.handler.Handle()
		}()
	}
}

func (server *tcp) Stop() error {

}

