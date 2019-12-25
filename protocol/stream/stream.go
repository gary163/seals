package stream

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"

	"github.com/gary163/seals/protocol"
)

const (
	defaultPackHead  = 2
	defaultByteOrder = "littleEndian"
)

var ErrIOReadWriterNil = errors.New("io.ReadWriter is nil")

type Stream struct {
	maxSend    int    //最大发送的长度
	maxRecv    int    //最小发送的长度
	n          int    //包头占用的字节数 可配置 1,2,4,8
	headEncode func([]byte, int)
	headDecode func([]byte) int
	littleEndian  bool //是否小端
}

type codec struct {
	protocol   *Stream
	headBuf    []byte //包长度字节
	bodyBuf    []byte //包体
	rw         io.ReadWriter //io接口
}

//初始化配置
func (s *Stream) Config(config string) error {
	cfg := make(map[string]string)
	json.Unmarshal([]byte(config), &cfg)

	if _,ok := cfg["n"]; !ok {
		cfg["n"] = strconv.Itoa(defaultPackHead)
	}
	if _,ok := cfg["byteOrder"]; !ok {
		cfg["byteOrder"] = defaultByteOrder
	}
	if _,ok := cfg["maxSend"]; !ok {
		cfg["maxSend"] = strconv.Itoa(0)
	}
	if _,ok := cfg["maxRecv"]; !ok {
		cfg["maxRecv"] = strconv.Itoa(0)
	}
	if _,ok := cfg["byteOrder"]; !ok {
		cfg["byteOrder"] = defaultByteOrder
	}

	n,_ := strconv.Atoi(cfg["n"])
	configNMap := map[int]int{1:1, 2:1, 4:1, 8:1}
	if _,ok := configNMap[n]; !ok {
		return errors.New("streamProtocol: n is invalid ")
	}
	s.n = n
	s.maxRecv,_ = strconv.Atoi(cfg["maxRecv"])
	s.maxSend,_ = strconv.Atoi(cfg["maxSend"])
	if cfg["byteOrder"] == "littleEndian" {
		s.littleEndian = true
	}else{
		s.littleEndian = false
	}

	return s.fixLen()
}

func (s *Stream) fixLen() error {
	switch s.n {
	case 1:
		if s.maxSend > math.MaxInt8 || s.maxSend == 0 {
			s.maxSend = math.MaxInt8
		}
		if s.maxRecv > math.MaxInt8 || s.maxRecv == 0 {
			s.maxRecv= math.MaxInt8
		}
		s.headEncode = func(b []byte, size int) {
			b[0] = byte(size)
		}
		s.headDecode = func(b []byte) int {
			return int(b[0])
		}
	case 2:
		if s.maxSend > math.MaxInt16 || s.maxSend == 0 {
			s.maxSend = math.MaxInt16
		}
		if s.maxRecv > math.MaxInt16 || s.maxRecv == 0 {
			s.maxRecv= math.MaxInt16
		}
		s.headEncode = func(b []byte, size int) {
			if s.littleEndian {
				binary.LittleEndian.PutUint16(b, uint16(size))
			}else{
				binary.BigEndian.PutUint16(b, uint16(size))
			}
		}
		s.headDecode = func(b []byte) int {
			if s.littleEndian {
				return int(binary.LittleEndian.Uint16(b))
			}else{
				return int(binary.BigEndian.Uint16(b))
			}
		}
	case 4:
		if s.maxSend > math.MaxInt32 || s.maxSend == 0 {
			s.maxSend = math.MaxInt32
		}
		if s.maxRecv > math.MaxInt32 || s.maxRecv == 0 {
			s.maxRecv= math.MaxInt32
		}
		s.headEncode = func(b []byte, size int) {
			if s.littleEndian {
				binary.LittleEndian.PutUint32(b, uint32(size))
			}else{
				binary.BigEndian.PutUint32(b, uint32(size))
			}
		}
		s.headDecode = func(b []byte) int {
			if s.littleEndian {
				return int(binary.LittleEndian.Uint32(b))
			}else{
				return int(binary.BigEndian.Uint32(b))
			}
		}
	case 8:
		if s.maxSend > math.MaxInt64 || s.maxSend == 0 {
			s.maxSend = math.MaxInt64
		}
		if s.maxRecv > math.MaxInt64 || s.maxRecv == 0 {
			s.maxRecv= math.MaxInt64
		}
		s.headEncode = func(b []byte, size int) {
			if s.littleEndian {
				binary.LittleEndian.PutUint64(b, uint64(size))
			}else{
				binary.BigEndian.PutUint64(b, uint64(size))
			}
		}
		s.headDecode = func(b []byte) int {
			if s.littleEndian {
				return int(binary.LittleEndian.Uint64(b))
			}else{
				return int(binary.BigEndian.Uint64(b))
			}
		}
	default:
		return errors.New("streamProtocol:pack head config is invaild")
	}
	return nil
}


func (s *Stream) NewCodec(rw io.ReadWriter) (protocol.Codec, error) {
	if rw == nil {
		return nil,errors.New("streamProtocpl: io.ReadWriter is nil")
	}
	var head [8]byte
	codec := &codec{
		protocol :s,
		rw:rw,
		headBuf:head[:s.n],
	}
	return codec,nil
}

func (c *codec) Receive() (interface{}, error) {
	if c.rw == nil {
		return nil,ErrIOReadWriterNil
	}
	if _, err := io.ReadFull(c.rw, c.headBuf); err != nil {
		return nil, err
	}
	size := c.protocol.headDecode(c.headBuf)
	if size > c.protocol.maxRecv {
		return nil, errors.New("streamProtocol:pack size is large than maxRecv")
	}
	if cap(c.bodyBuf) < size {
		c.bodyBuf = make([]byte, size, size+128)
	}
	buff := c.bodyBuf[:size]
	if _, err := io.ReadFull(c.rw, buff); err != nil {
		return nil, err
	}
	return buff, nil
}

func (c *codec) Send(msg interface{}) error {
	if c.rw == nil {
		return ErrIOReadWriterNil
	}
	val, ok := msg.([]byte)
	if !ok {
		return errors.New("streamProtocol:method Send Parameter type error")
	}

	sendBuf := new(bytes.Buffer)
	sendBuf.Write(c.headBuf)
	sendBuf.Write(val)
	buf := sendBuf.Bytes()
	c.protocol.headEncode(buf, len(val))
	_,err := c.rw.Write(buf)
	return err
}


func (c *codec) Close() error {
	if closer, ok := c.rw.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func init() {
	protocol.Register("stream", &Stream{})
}
