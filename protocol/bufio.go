package protocol

import (
	"bufio"
	"fmt"
	"io"
)

const (
	defaultReadSize  = 128
 	defaultWriteSize = 128
)

type bufioProtocol struct {
	base      Protocol
	readSize  int
	writeSize int
}

func newBufio(base Protocol, readSize, writeSize int) (Protocol,error) {
	if readSize < defaultReadSize {
		readSize = defaultReadSize
	}
	if writeSize < defaultWriteSize {
		writeSize = defaultWriteSize
	}

	bp := &bufioProtocol{}
	bp.base = base
	bp.readSize = readSize
	bp.writeSize = writeSize
	return bp, nil
}

func (bio *bufioProtocol) Register(interface{}){}

func (bio *bufioProtocol) NewCodec(rw io.ReadWriter) (code Codec,err error) {
	codec := &bufioCodec{}
	codec.stream.Reader = bufio.NewReaderSize(rw,bio.readSize)
	codec.stream.w = bufio.NewWriterSize(rw,bio.writeSize)
	codec.stream.Writer = codec.stream.w
	codec.stream.c,_ = rw.(io.Closer)

	codec.base, err = bio.base.NewCodec(&codec.stream)
	if err != nil {
		return nil ,err
	}

	return codec, nil
}

type bufioStream struct {
	io.Reader
	io.Writer
	c io.Closer
	w *bufio.Writer
}

func (s *bufioStream) Flush() error {
	if s.w != nil {
		return s.w.Flush()
	}
	return nil
}

func (s *bufioStream) close() error {
	if s.c != nil {
		fmt.Println("bufio close....")
		return s.c.Close()
	}
	return nil
}

type bufioCodec struct {
	base Codec
	stream bufioStream
}

func (c *bufioCodec) Send(msg interface{}) error {
	if err := c.base.Send(msg); err != nil {
		return err
	}
	return c.stream.Flush()
}

func (c *bufioCodec) Receive() (interface{}, error) {
	return c.base.Receive()
}

func (c *bufioCodec) Close() error {
	err1 := c.base.Close()
	err2 := c.stream.close()
	if err1 != nil {
		return err1
	}
	return err2
}





