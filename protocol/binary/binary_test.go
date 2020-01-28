package binary

import (
	"bytes"
	"testing"

	"github.com/gary163/seals/protocol"
)

func TestStream(t *testing.T) {
	var buff  bytes.Buffer
	stream, err := protocol.NewProtocol("binary",`{"fixlen":{},"bufio":{}}`)
	if err != nil {
		t.Errorf("NewProtocol err:%v\n",err)
	}
	codec, _ := stream.NewCodec(&buff)
	msg := []byte{'y','k','f','1','2','3'}
	err = codec.Send(msg)
	if err != nil {
		t.Errorf("send Error:%v\n",err)
	}

	data,err := codec.Receive()
	if err != nil {
		t.Errorf("receive err:%v\n",err)
	}
	t.Logf("recv:%s\n",data)
}
