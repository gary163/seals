package stream

import (
	"io"

	"github.com/gary163/seals/protocol"
)


type binary struct {
}

type codec struct {
}

func (s *binary) Register(interface{}){}

func (s *binary) NewCodec(rw io.ReadWriter) (protocol.Codec, error) {
	return nil,nil
}

func (c *codec) Receive() (interface{}, error) {
	return nil, nil
}

func (c *codec) Send(msg interface{}) error {
	return nil
}

func (c *codec) Close() error {
	return nil
}

func init() {
	protocol.Register("binary", &binary{})
}
