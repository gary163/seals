package stream

import (
	"bytes"
	"testing"

	"github.com/gary163/seals/protocol"
)

func TestStream(t *testing.T) {
	var buff  bytes.Buffer
	stream, err := protocol.NewProtocol("stream","")
	if err != nil {
		t.Errorf("NewProtocol err:%v\n",err)
	}
	stream.SetIOReadWriter(&buff)
	msg := []byte{'y','k','f','1','2','3'}
	err = stream.Send(msg)
	if err != nil {
		t.Errorf("send Error:%v\n",err)
	}


	data,err := stream.Receive()
	if err != nil {
		t.Errorf("receive err:%v\n",err)
	}
	t.Logf("recv:%s\n",data)
}
