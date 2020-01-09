package codec

import (
	"bytes"
	"testing"

	"github.com/gary163/seals/protocol"
)

type Member struct {
	Name string
	Age int
}

func JsonTest( t *testing.T, protocol protocol.Protocol){
	var stream bytes.Buffer
	codec,err := protocol.NewCodec(&stream)
	if err != nil {
		t.Fatalf("New codec err:%v\n",err)
	}
	defer codec.Close()

	protocol.Register(&Member{})
	sendMsg := Member{"gary",18}
	err = codec.Send(&sendMsg)
	if err != nil {
		t.Fatalf("Send msg err:%v\n",err)
	}
	recvmsg,err := codec.Receive()

	if err != nil {
		t.Fatal(err)
	}

	if _,ok := recvmsg.(*Member); !ok {
		t.Fatalf("message type not match: %#v",recvmsg)
	}

	if sendMsg != *(recvmsg.(*Member)) {
		t.Fatalf("message not match: %v, %v", sendMsg, recvmsg)
	}
	//t.Logf("recvMsg:%#v",recvmsg)
}

func TestJson(t *testing.T) {
	protocolJson,_ := protocol.NewProtocol("json","")
	JsonTest( t,protocolJson)
	protocolJsonAndFixlen,_ := protocol.NewProtocol("json",`{"fixlen":{}}`)
	JsonTest( t,protocolJsonAndFixlen)
	protocolJsonBufio,_ := protocol.NewProtocol("json",`{"bufio":{}}`)
	JsonTest( t,protocolJsonBufio)
	protocolJsonBufioAndFixlen,_ := protocol.NewProtocol("json",`{"fixlen":{},"bufio":{}}`)
	JsonTest( t,protocolJsonBufioAndFixlen)
}

