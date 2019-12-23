package tcpserver

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/gary163/seals/protocol"
	_ "github.com/gary163/seals/protocol/stream"
	"github.com/gary163/seals/server"
)

type TestProtocol struct {
	rw io.ReadWriteCloser
}

func (p *TestProtocol) Send(msg interface{}) error {
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

func (p *TestProtocol) Receive() (interface{},error) {
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

func (p *TestProtocol) Close() error {
	return p.rw.Close()
}

func NewTestProtocol (rw io.ReadWriter) (*TestProtocol,error) {
	p := &TestProtocol{}
	p.rw = rw.(io.ReadWriteCloser)
	return p,nil
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

	client,err := server.NewClient("tcpClient",`{"addr":"127.0.0.1:55567","connNum":"100"}`,proto, &tcpclient{t})
	if err != nil {
		t.Fatalf("New Client err:%v\n",err)
	}


	client.Run()

	//client.Close()

	fmt.Printf("After Client Run......")
	srv.Stop()
	//client.Close()
}

func TestServerAsync(t *testing.T){
	TcpServerTest(t,100)
}

func TestServerSync(t *testing.T){
	TcpServerTest(t,0)
}

type tcpserver struct{
	t *testing.T
}

func (s *tcpserver) Handle(session *server.Session) {
	//s.t.Log("Server Handling ........")
	for {
		recv,err := session.Receive()
		//s.t.Logf("Server receive msg:%v\n",recv)
		//fmt.Printf("Server Handle Recv:%v\n",recv)
		if err != nil {
			s.t.Logf("Server Handle Recv err :%v\n",err)
			return
		}
		if err = session.Send(recv); err != nil {
			fmt.Printf("Server Handle Send err:%v\n",err)
			return
		}
		//fmt.Printf("Server send msg:%v\n",recv)
		//s.t.Logf("Server send msg:%v\n",recv)
	}
}

type tcpclient struct{
	t *testing.T
}


var counter int32

func (c *tcpclient) Handle(session *server.Session) {

	atomic.AddInt32(&counter,1)
	//for i:=0; i<100; i++ {
		//c.t.Log("Client handling.....")
		msg1 := RandByte(2000)
		//msg1 := []byte("garyyangygyubb7867887897896ftuggbv")
		//c.t.Logf("Client send msg:%v\n",msg1)
		//fmt.Printf("Client send msg:%v\n",msg1)
		fmt.Printf("Send client[%d]:%v,len(msg1):%v\n",session.ID(),msg1[0],len(msg1))

		err := session.Send(msg1)
		recv,err := session.Receive()
		if err != nil {
			return
		}
		msg2 := recv.([]byte)
		//fmt.Printf("Client receive msg:%v\n",msg2)
		fmt.Printf("Recv client[%d]:%v,len(msg2):%v\n",session.ID(),msg2[0],len(msg2))
		//c.t.Logf("Client receive msg:%v\n",msg2)
		if ok := bytes.Equal(msg1, msg2); !ok {
			c.t.Errorf("msg1(%s) not equal msg2(%s)\n",msg1,msg2)
		}
		//fmt.Println("Client receive endding.....")
	//}

}

