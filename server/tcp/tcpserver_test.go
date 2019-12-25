package tcpserver

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gary163/seals/protocol"
	_ "github.com/gary163/seals/protocol/stream"
	"github.com/gary163/seals/server"
	"io"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

type TestProtocol struct {
}

func (p *TestProtocol) Config(config string) error {
	return nil
}

func (p *TestProtocol) NewCodec(rw io.ReadWriter) (protocol.Codec, error) {
	codec := &TestCodec{
		rw:rw.(io.ReadWriteCloser),
		protocol:p,
	}
	return codec,nil
}

type TestCodec struct {
	rw io.ReadWriteCloser
	protocol *TestProtocol
}


func (p *TestCodec) Send(msg interface{}) error {
	val,ok := msg.([]byte)
	if !ok {
		return errors.New("Send:msg type error")
	}
	l := len(val)
	buff := make([]byte,2)
	binary.LittleEndian.PutUint16(buff,uint16(l))
	_,err := p.rw.Write(buff)
	if err != nil {
		return err
	}
	_,err = p.rw.Write(val)
	if err != nil {
		return err
	}
	return nil
}

func (p *TestCodec) Receive() (interface{},error) {
	var head [2]byte
	_,err := io.ReadFull(p.rw,head[:])
	if err != nil {
		return nil, err
	}
	len := binary.LittleEndian.Uint16(head[:])

	body := make([]byte,len)
	if _,err := io.ReadFull(p.rw,body); err != nil {
		return nil,err
	}
	return body,nil
}

func (p *TestCodec) Close() error {
	return p.rw.Close()
}


func RandByte(n int) []byte {
	n = rand.Intn(n)+1
	bytes := make([]byte,n)
	for i := 0; i<n; i++ {
		bytes[i] = byte(rand.Intn(255))
	}
	return bytes
}


func TcpServerTest(t *testing.T ,sendChanSize int) {
	cfg := make(map[string]string)
	cfg["addr"] = "0.0.0.0:55567"
	cfg["sendChanSize"]  = strconv.Itoa(sendChanSize)
	strCfg,err := json.Marshal(cfg)

	proto, err := protocol.NewProtocol("stream",`{"n":"2"}`)
	//proto := &TestProtocol{}
	if err != nil {
		t.Fatalf("protcol error:%v\n",err)
	}

	srv,err := server.NewServer("tcpServer",string(strCfg),proto, &tcpserver{t})
	if err != nil {
		t.Fatalf("New Server error:%v\n",err)
	}

	go func(){
		srv.Run()
	}()

	client,err := server.NewClient("tcpClient",`{"addr":"127.0.0.1:55567","connNum":"1"}`,proto, &tcpclient{t})
	if err != nil {
		t.Fatalf("New Client err:%v\n",err)
	}

	client.Run()
	srv.Stop()
	//client.Close()
}

func TestServerAsync(t *testing.T){
	TcpServerTest(t,100)
}

func TestServerSync(t *testing.T){
	TcpServerTest(t,0)
}

func TestServerWithCallbackFunc(t *testing.T){
	testCallback = true
	TcpServerTest(t,1024)
}

type tcpserver struct{
	t *testing.T
}

var testCallback = false

func (s *tcpserver) Handle(session *server.Session) {
	if testCallback {
		callbackChan := make(chan int, 10)
		for i:=0; i<10; i++ {
			func(i int){
				callback := func(){
					callbackChan <- i
				}
				session.AddCloseCallback(nil,i,callback)
				session.DelCloseCallback(nil,i)
				session.AddCloseCallback(nil,i,callback)
			}(i)
		}

		defer func(){
			for i:=0; i<10; i++ {
				n := <- callbackChan
				s.t.Logf("callback Chan n:%d\n",n)
				if i != n {
					s.t.Fatalf("i:%d not equal recv n:%d\n",i,n)
				}
			}
		}()
	}

	for {
		recv,err := session.Receive()
		if err != nil {
			s.t.Logf("Server Handle Recv err :%v\n",err)
			return
		}
		if err = session.Send(recv); err != nil {
			return
		}
	}
}

type tcpclient struct{
	t *testing.T
}

func (c *tcpclient) Handle(session *server.Session) {
	defer session.Close()
	for i:=0; i<100; i++ {
		msg1 := RandByte(2000)
		err := session.Send(msg1)
		recv,err := session.Receive()
		if err != nil {
			return
		}
		msg2 := recv.([]byte)
		if ok := bytes.Equal(msg1, msg2); !ok {
			c.t.Errorf("msg1(%s) not equal msg2(%s)\n",msg1,msg2)
		}
	}
}

const clientConnNum = 5
const msgNum = 1000
var channelMsgs = make([][]byte, msgNum)
var waitClient sync.WaitGroup
var channel = server.NewChannel()

func sendToclient(){
	waitClient.Wait()
	fmt.Println("SendToClient .......")
	for i:=0; i<msgNum;i++ {
		msg := RandByte(500)
		channelMsgs[i] = msg
		channel.Fetch(func(session *server.Session){
			err := session.Send(msg)
			fmt.Printf("SendToClient session[%d], err:%v\n",session.ID(),err)
		})
	}
}

type channelServer struct {
	t *testing.T
}

func (cs *channelServer) Handle(session *server.Session) {
	//defer session.Close()
	fmt.Printf("Server [%d] Handling...\n",session.ID())
	channel.Set(session.ID(),session)
	channel.Delete(session.ID())
	if channel.Get(session.ID()) != nil {
		cs.t.Fatal("After set and get session is not nil")
	}
	channel.Set(session.ID(),session)
}

type channelClient struct {
	t *testing.T
}

func (cc *channelClient) Handle(session *server.Session){
	waitClient.Done()
	fmt.Printf("Client[%d] Handling\n",session.ID())
	for i:=0; i<msgNum; i++ {
		recv,_ := session.Receive()
		msg := recv.([]byte)
		fmt.Printf("Client Recv len:%d\n",len(msg))
		if equal := bytes.Equal(msg,channelMsgs[i]); !equal {
			cc.t.Fatalf("Recv msg:%v not equal send msg:%v\n ",msg,channelMsgs[i])
		}
	}
}

func TestChannel(t *testing.T){
	waitClient.Add(clientConnNum)
	proto, err := protocol.NewProtocol("stream","")
	if err != nil {
		t.Fatalf("New protocol err:%v\n",err)
	}
	srv, err := server.NewServer("tcpServer",`{"addr":"0.0.0.0:58567"}`,proto,&channelServer{t})
	if err != nil {
		t.Fatalf("New server err:%v\n",err)
	}
	go func(){
		fmt.Println("Server running......")
		srv.Run()
	}()

	go sendToclient()

	cli, err := server.NewClient("tcpClient",`{"addr":"127.0.0.1:58567","connNum":"5"}`, proto, &channelClient{t})
	if err != nil {
		t.Fatalf("New client err:%v\n",err)
	}
	fmt.Println("Client running......")
	time.Sleep(time.Second)
	cli.Run()
	srv.Stop()
}

