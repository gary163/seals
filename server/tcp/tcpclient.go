package tcpserver

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gary163/seals/protocol"
	"github.com/gary163/seals/server"
)

const (
	defaultConnNum    = 1
	defaultMaxTryTime = 3
)

type tcpClient struct {
	addr         string
	timeout      int
	sendChanSize int
	handler      server.Handler
	protocol     protocol.Protocol
	sm           *server.SessionManager
	connNum      int
	wg           sync.WaitGroup
}

func (c *tcpClient) Init(config string, protocol protocol.Protocol, handler server.Handler, sm *server.SessionManager) error {
	var cfg map[string]string
	json.Unmarshal([]byte(config),&cfg)

	if _,ok := cfg["addr"]; !ok {
		return errors.New("TcpClient:Missing addr parameter")
	}
	if _,ok := cfg["timeout"]; !ok {
		cfg["timeout"] = strconv.Itoa(0)
	}
	if _,ok := cfg["sendChanSize"]; !ok {
		cfg["sendChanSize"] = strconv.Itoa(defaultSendChanSize)
	}
	if _,ok := cfg["connNum"]; !ok {
		cfg["connNum"] = strconv.Itoa(defaultConnNum)
	}

	c.addr           = cfg["addr"]
	c.timeout,_      = strconv.Atoi(cfg["timeout"])
	c.sendChanSize,_ = strconv.Atoi(cfg["sendChanSize"])
	c.connNum,_= strconv.Atoi(cfg["connNum"])
	c.handler  = handler
	c.protocol = protocol
	c.sm       = sm
	return nil
}

func (c *tcpClient) Run() {
	for i:=0; i<c.connNum; i++ {
		c.wg.Add(1)
		go func(){
			conn,err := c.dial()
			if err != nil {
				log.Fatalf("Client dial got a err:%v\n",err)
			}
			codec,_ := c.protocol.NewCodec(conn)
			session := c.sm.NewSession(codec, c.sendChanSize)
			c.handler.Handle(session)
			c.wg.Done()
		}()
	}
	c.wg.Wait()
}

func (c *tcpClient) dial() (net.Conn,error) {
	var netConn net.Conn
	var err error
	tryConnTime := 0
	if c.timeout > 0 {
		for{
			netConn,err = net.DialTimeout("tcp",c.addr,time.Duration(c.timeout))
			if err == nil || tryConnTime > defaultMaxTryTime{
				break
			}
			time.Sleep(50*time.Millisecond)
			tryConnTime++
		}
	}else{
		for{
			netConn, err = net.Dial("tcp", c.addr)
			if err == nil || tryConnTime > defaultMaxTryTime {
				break
			}
			time.Sleep(50*time.Millisecond)
			tryConnTime++
		}
	}

	return netConn,err
}

func (c *tcpClient) Close() error {
	 c.sm.Destroy()
	 return nil
}

func init(){
	server.RegisterClient("tcpClient", &tcpClient{})
}