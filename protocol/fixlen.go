package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

const (
	defaultPackHead  = 2
)

var (
	defaultByteOrder = binary.BigEndian
)

var ErrIOReadWriterNil = errors.New("io.ReadWriter is nil")

type fixlenProtocol struct {
	maxSend    int    //最大发送的长度
	maxRecv    int    //最小发送的长度
	n          int    //包头占用的字节数 可配置 1,2,4,8
	headEncode func([]byte, int)
	headDecode func([]byte) int
	byteOrder  binary.ByteOrder
	base       Protocol
}

func newFixlenProtocol(base Protocol,maxSend,maxRecv,n int, byteOrder binary.ByteOrder) (Protocol, error) {
	fix := &fixlenProtocol{}
	fix.base = base
	fix.maxRecv = maxRecv
	fix.maxSend = maxSend
	fix.n       = n
	fix.byteOrder = byteOrder
	if err := fix.fixLen(); err != nil {
		return nil, err
	}
	return fix, nil
}

func (fix *fixlenProtocol) fixLen() error {
	switch fix.n {
	case 1:
		if fix.maxSend > math.MaxInt8 || fix.maxSend == 0 {
			fix.maxSend = math.MaxInt8
		}
		if fix.maxRecv > math.MaxInt8 || fix.maxRecv == 0 {
			fix.maxRecv= math.MaxInt8
		}
		fix.headEncode = func(b []byte, size int) {
			b[0] = byte(size)
		}
		fix.headDecode = func(b []byte) int {
			return int(b[0])
		}
	case 2:
		if fix.maxSend > math.MaxInt16 || fix.maxSend == 0 {
			fix.maxSend = math.MaxInt16
		}
		if fix.maxRecv > math.MaxInt16 || fix.maxRecv == 0 {
			fix.maxRecv= math.MaxInt16
		}
		fix.headEncode = func(b []byte, size int) {
			fix.byteOrder.PutUint16(b, uint16(size))
		}
		fix.headDecode = func(b []byte) int {
			return int(fix.byteOrder.Uint16(b))
		}
	case 4:
		if fix.maxSend > math.MaxInt32 || fix.maxSend == 0 {
			fix.maxSend = math.MaxInt32
		}
		if fix.maxRecv > math.MaxInt32 || fix.maxRecv == 0 {
			fix.maxRecv= math.MaxInt32
		}
		fix.headEncode = func(b []byte, size int) {
			fix.byteOrder.PutUint32(b, uint32(size))
		}
		fix.headDecode = func(b []byte) int {
			return int(fix.byteOrder.Uint32(b))
		}
	case 8:
		if fix.maxSend > math.MaxInt64 || fix.maxSend == 0 {
			fix.maxSend = math.MaxInt64
		}
		if fix.maxRecv > math.MaxInt64 || fix.maxRecv == 0 {
			fix.maxRecv= math.MaxInt64
		}
		fix.headEncode = func(b []byte, size int) {
			fix.byteOrder.PutUint64(b, uint64(size))
		}
		fix.headDecode = func(b []byte) int {
			return int(fix.byteOrder.Uint64(b))
		}
	default:
		return errors.New("streamProtocol:pack head config is invaild")
	}
	return nil
}

func (fix *fixlenProtocol) Register(interface{}){}

func (fix *fixlenProtocol) NewCodec(rw io.ReadWriter) (Codec, error) {
	c := &codec{}
	c.protocol = fix
	c.rw = rw
	var head [8]byte
	c.headBuf = head[:fix.n]

	codec,err := fix.base.NewCodec(&c.streamReadWriter)
	if err != nil {
		return nil, errors.New("Try to new base codec and  got a error")
	}

	c.base = codec
	return c, nil
}

type codec struct {
	protocol    *fixlenProtocol
	headBuf    []byte //包长度字节
	bodyBuf    []byte //包体
	rw         io.ReadWriter //io接口
	base       Codec
	streamReadWriter
}

type streamReadWriter struct {
	sendBuf bytes.Buffer
	recvBuf bytes.Reader
}

func (s *streamReadWriter) Read(p []byte) (int, error) {
	return s.recvBuf.Read(p)
}

func (s *streamReadWriter) Write(p []byte) (int, error) {
	return s.sendBuf.Write(p)
}

func (c *codec) Receive() (interface{}, error) {
	head := c.headBuf
	if _,err := io.ReadFull(c.rw,head); err != nil {
		return nil, err
	}

	size := c.protocol.headDecode(head)
	if size > c.protocol.maxRecv {
		return nil, errors.New("pack size is large than maxRecv")
	}

	if cap(c.bodyBuf) < size {
		c.bodyBuf = make([]byte,size+128)
	}

	body := c.bodyBuf[:size]
	if _, err := io.ReadFull(c.rw, body); err != nil {
		return nil, err
	}

	c.recvBuf.Reset(body)
	if c.base == nil {
		return body, nil
	}
	return c.base.Receive()
}

func (c *codec) Send(msg interface{}) error {
	c.sendBuf.Reset()
	c.sendBuf.Write(c.headBuf)
	if c.base != nil {
		if err := c.base.Send(msg); err != nil {
			return nil
		}
	} else {
		val, ok := msg.([]byte)
		if !ok {
			return errors.New("Send Parameter type error")
		}
		c.sendBuf.Write(val)
	}

	buff := c.sendBuf.Bytes()
	c.protocol.headEncode(buff, len(buff) - c.protocol.n)
	_, err := c.rw.Write(buff)
	return err
}

func (c *codec) Close() error {
	if closer, ok := c.rw.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}



