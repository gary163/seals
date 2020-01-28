package protobuf

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"testing"

	"github.com/gary163/seals/protocol"
	"github.com/gary163/seals/protocol/protobuf/example"
)


func ProtobufTest( t *testing.T, protocol protocol.Protocol){
	var stream bytes.Buffer
	codec,err := protocol.NewCodec(&stream)
	if err != nil {
		t.Fatalf("New codec err:%v\n",err)
	}
	defer codec.Close()

	protocol.Register(&example.Test{})
	sendMsg := &example.Test{
		// 使用辅助函数设置域的值
		Label: proto.String("hello"),
		Type:  proto.Int32(17),
		Optionalgroup: &example.Test_OptionalGroup{
			RequiredField: proto.String("good bye"),
		},
	}
	err = codec.Send(sendMsg)
	if err != nil {
		t.Fatalf("Send msg err:%v\n",err)
	}
	recvmsg,err := codec.Receive()

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("recvMsg:%#v",recvmsg)
}

func TestProtobuf(t *testing.T) {
	protocolJson,_ := protocol.NewProtocol("protobuf","")
	ProtobufTest( t,protocolJson)
	/*
	protocolJsonAndFixlen,_ := protocol.NewProtocol("protobuf",`{"fixlen":{}}`)
	JsonTest( t,protocolJsonAndFixlen)
	protocolJsonBufio,_ := protocol.NewProtocol("protobuf",`{"bufio":{}}`)
	JsonTest( t,protocolJsonBufio)
	protocolJsonBufioAndFixlen,_ := protocol.NewProtocol("protobuf",`{"fixlen":{},"bufio":{}}`)
	JsonTest( t,protocolJsonBufioAndFixlen)
	*/
}

