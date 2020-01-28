package websocket

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"


	"github.com/gary163/seals/protocol"
	"github.com/gary163/seals/server"
	"github.com/gobwas/ws"
	"github.com/gorilla/websocket"

)

const (
	defaultMaxConn = 200000
	defaultSendChanSize = 1024
	maxTryTime = 3
	defaultAddr = "0.0.0.0:0"
	defaultHttpTimeout = 5
)

type WSServer struct {
	addr         string
	maxConn      int//最大连接数
	listener     net.Listener
	sendChanSize int//异步send的buffer个数
	protocol     protocol.Protocol
	handler      server.Handler
	wsHandler    *WSHandler
	sm           *server.SessionManager
	httpTimeout  time.Duration
	certFile     string
	keyFile      string
}

type WSHandler struct {
	maxConn     int
	upgrader    websocket.Upgrader
	server      *WSServer
}

func init() {
	server.RegisterServer("websocketServer",&WSServer{})
}

func (s *WSServer) Init(config string, protocol protocol.Protocol, handler server.Handler, sm *server.SessionManager) error {
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

	if _,ok := cfg["httpTimeout"]; !ok {
		cfg["httpTimeout"] = strconv.Itoa(int(defaultHttpTimeout * time.Second))
	}

	if _,ok := cfg["certFile"]; !ok {
		cfg["certFile"] = ""
	}

	if _,ok := cfg["keyFile"]; !ok {
		cfg["keyFile"] = ""
	}

	s.maxConn,_      = strconv.Atoi(cfg["maxConn"])
	s.sendChanSize,_ = strconv.Atoi(cfg["sendChanSize"])
	s.addr,_         = cfg["addr"]
	httpTimeout,_    := strconv.Atoi(cfg["httpTimeout"])
	s.httpTimeout =  time.Duration(httpTimeout)
	s.certFile,_      = cfg["certFile"]
	s.keyFile,_       = cfg["keyFile"]
	s.protocol = protocol
	s.handler  = handler
	s.sm       = sm

	listener, err := net.Listen("tcp", s.addr)
	if s.certFile != "" || s.keyFile != "" {
		config := &tls.Config{}
		config.NextProtos = []string{"http/1.1"}

		var err error
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(s.certFile, s.keyFile)
		if err != nil {
			return err
		}

		s.listener = tls.NewListener(listener, config)
	}

	s.wsHandler = &WSHandler{
		maxConn:      s.maxConn,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: s.httpTimeout,
			CheckOrigin:      func(_ *http.Request) bool { return true },
		},
		server:s,
	}

	httpServer := &http.Server{
		Addr:           s.addr,
		Handler:        s.wsHandler,
		ReadTimeout:    s.httpTimeout,
		WriteTimeout:   s.httpTimeout,
		MaxHeaderBytes: 1024,
	}

	go httpServer.Serve(listener)

	return nil
}

func (s *WSServer) Run() error {
	tryTime := 0
	for{
		conn,err := s.listener.Accept()
		if s.sm.Len() > int64(s.maxConn) {
			log.Printf("Too manay connection:%d\n",s.maxConn)
			conn.Close()
			continue
		}

		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() && tryTime < maxTryTime{
				time.Sleep(50*time.Millisecond)
				tryTime ++
				continue
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				return io.EOF
			}
		}

		_, err = ws.Upgrade(conn)
		if err != nil {
			// handle error
		}

		go func() {
			codec,_ := s.protocol.NewCodec(conn)
			session := s.sm.NewSession(codec, s.sendChanSize)
			s.handler.Handle(session)
		}()


		go func(){
			codec,_ := s.protocol.NewCodec(conn)
			session := s.sm.NewSession(codec, s.sendChanSize)
			s.handler.Handle(session)
		}()
	}
}

func (s *WSServer) Stop() error {
	s.listener.Close()
	s.sm.Destroy()
	return nil
}

