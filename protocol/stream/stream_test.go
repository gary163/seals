package stream

import (
	"bytes"
	"io"
	"testing"

	"github.com/gary163/seals/protocol"
)

func TestStream(t *testing.T) {
	var buff  bytes.Buffer
	stream, _ := protocol.NewProtocol("stream",``)
	stream.SetIOReadWriter(buff)
	msg := []byte{'y','k','f','1','2','3'}
	err := stream.Send(msg)
	if err != nil {
		t.Errorf("send Error:%v\n",err)
	}


	data,err := stream.Receive()
	if err != nil {
		t.Errorf("receive err:%v\n",err)
	}
	t.Logf("recv:%s\n",data)
}
