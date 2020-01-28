package binary

import (
	"errors"
	"io"
	"io/ioutil"

	"github.com/gary163/seals/protocol"
)


type binary struct {
}

func (s *binary) Register(interface{}){}

func (s *binary) NewCodec(rw io.ReadWriter) (protocol.Codec, error){
	return &binaryCodec{rw}, nil
}

type binaryCodec struct {
	rw io.ReadWriter
}

func (c *binaryCodec) Receive() (interface{}, error) {
	recv,err := ioutil.ReadAll(c.rw)
	return recv,err
}

func (c *binaryCodec) Send(msg interface{}) error {
	val, ok := msg.([]byte)
	if !ok {
		return errors.New("cannot be converted []byte")
	}
	_,err := c.rw.Write(val)
	return err
}

func (c *binaryCodec) Close() error {
	if closer := c.rw.(io.Closer); closer != nil {
		return closer.Close()
	}
	return nil
}

func init() {
	protocol.Register("binary", &binary{})
}
