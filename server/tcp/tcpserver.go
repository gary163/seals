package tcpserver

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gary163/seals/protocol"
	"github.com/gary163/seals/server"
)

const (
	defaultMaxConn = 200000
	defaultSendChanSize = 1024
	maxTryTime = 3
	defaultAddr = "0.0.0.0:0"
)

type tcpServer struct {
	addr         string
	maxConn      int//最大连接数
	listener     net.Listener
	sendChanSize int//异步send的buffer个数
	protocol     protocol.Protocol
	handler      server.Handler
	sm           *server.SessionManager
}

func init() {
	server.RegisterServer("tcpServer",&tcpServer{})
}

func (s *tcpServer) Init(config string, protocol protocol.Protocol, handler server.Handler, sm *server.SessionManager) error {
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

	if _,ok := cfg["addr"]; !ok {
		cfg["addr"] = defaultAddr
	}

	s.maxConn,_      = strconv.Atoi(cfg["maxConn"])
	s.sendChanSize,_ = strconv.Atoi(cfg["sendChanSize"])
	s.addr,_         = cfg["addr"]
	s.protocol = protocol
	s.handler  = handler
	s.sm       = sm

	if s.listener,err = net.Listen("tcp", s.addr); err!= nil {
		return err
	}
	return nil
}

func (s *tcpServer) Run() error {
	tryTime := 0
	for{
		conn,err := s.listener.Accept()
		if s.sm.Len() > int64(defaultMaxConn) {
			log.Printf("Exceeded the maximum number of connections:%d\n",defaultMaxConn)
			conn.Close()
		}

		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() && tryTime < maxTryTime{
				time.Sleep(5*time.Millisecond)
				tryTime ++
				log.Printf("tryTime:%v\n",tryTime)
				continue
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				return io.EOF
			}
		}

		go func(){
			codec,_ := s.protocol.NewCodec(conn)
			session := s.sm.NewSession(codec, s.sendChanSize)
			s.handler.Handle(session)
		}()
	}
}

func (s *tcpServer) Stop() error {
	log.Printf("TcpServer stoping....")
	s.listener.Close()
	s.sm.Destroy()
	return nil
}

