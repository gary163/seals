package tcpserver

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/gary163/seals/protocol"
	"github.com/gary163/seals/server"
)

type client struct {
	addr         string
	timeout      int
	sendChanSize int
	handler      server.Handler
	protocol     protocol.Protocol
	conn         net.Conn
}

func (c *client) Init(config string, protocol protocol.Protocol, handler server.Handler) error {
	var cfg map[string]string
	json.Unmarshal([]byte(config),&cfg)

	if _,ok := cfg["addr"]; !ok {
		return errors.New("Missing addr parameter")
	}
	if _,ok := cfg["timeout"]; !ok {
		cfg["timeout"] = strconv.Itoa(0)
	}
	if _,ok := cfg["sendChanSize"]; !ok {
		cfg["sendChanSize"] = strconv.Itoa(defaultSendChanSize)
	}

	c.addr = cfg["addr"]
	c.timeout,_ = strconv.Atoi(cfg["timeout"])
	c.sendChanSize,_ = strconv.Atoi(cfg["sendChanSize"])
	c.handler  = handler
	c.protocol = protocol
	return nil
}

func (c *client) Dial() error {
	if c.timeout > 0 {
		conn,err := net.DialTimeout("tcp",c.addr,time.Duration(c.timeout))
		if err != nil {
			return err
		}
		c.conn = conn
	}else{
		conn,err := net.Dial("tcp",c.addr);
		if err != nil {
			return err
		}
		c.conn = conn
	}

	session := server.NewSession(c.protocol, c.sendChanSize)
	c.handler.Handle(session)
	return nil
}

func (c *client) Close() error {
	return c.conn.Close()
}