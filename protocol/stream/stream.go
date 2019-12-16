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

type stream struct {
	maxSend    int    //最大发送的长度
	maxRecv    int    //最小发送的长度
	headBuf    []byte //包长度字节
	bodyBuf    []byte //包体
	headEncode func([]byte, int)
	headDecode func([]byte) int
	rw         io.ReadWriter //io接口
	n          int           //包头占用的字节数 可配置 1,2,4,8
}

const DEFAULT_PACK_HEAD_N = 2

//初始化配置
func (s *stream) Init(rw io.ReadWriter, config string) error {
	if rw == nil {
		return errors.New("streamProtocol:rw is nil")
	}
	cfg := make(map[string]string)
	json.Unmarshal([]byte(config), &cfg)

	n, ok := cfg["n"]
	if !ok {
		s.n = DEFAULT_PACK_HEAD_N
	} else {
		s.n, _ = strconv.Atoi(n)
	}
	return s.fixLen()
}

func (s *stream) fixLen() error {
	var head [8]byte
	switch s.n {
	case 1:
		s.maxRecv = math.MaxInt8
		s.maxSend = math.MaxInt8
		s.headEncode = func(b []byte, size int) {
			b[0] = byte(size)
		}
		s.headDecode = func(b []byte) int {
			return int(b[0])
		}
		s.headBuf = head[:1]
	case 2:
		s.maxSend = math.MaxInt16
		s.maxRecv = math.MaxInt16
		s.headEncode = func(b []byte, size int) {
			binary.BigEndian.PutUint16(b, uint16(size))
		}
		s.headDecode = func(b []byte) int {
			return int(binary.LittleEndian.Uint16(b))
		}
		s.headBuf = head[:2]
	case 4:
		s.maxSend = math.MaxInt32
		s.maxRecv = math.MaxInt32
		s.headEncode = func(b []byte, size int) {
			binary.BigEndian.PutUint32(b, uint32(size))
		}
		s.headDecode = func(b []byte) int {
			return int(binary.LittleEndian.Uint32(b))
		}
		s.headBuf = head[:4]
	case 8:
		s.maxSend = math.MaxInt64
		s.maxRecv = math.MaxInt64
		s.headEncode = func(b []byte, size int) {
			binary.BigEndian.PutUint64(b, uint64(size))
		}
		s.headDecode = func(b []byte) int {
			return int(binary.LittleEndian.Uint64(b))
		}
		s.headBuf = head[:8]
	default:
		return errors.New("streamProtocol:pack head config is invaild")

	}
	return nil
}

func (s *stream) Receive() (interface{}, error) {
	if _, err := io.ReadFull(s.rw, s.headBuf); err != nil {
		return nil, err
	}
	size := s.headDecode(s.headBuf)
	if size > s.maxRecv {
		return nil, errors.New("streamProtocol:pack size is large than maxRecv")
	}
	if cap(s.bodyBuf) < size {
		s.bodyBuf = make([]byte, size*2)
	}
	buff := s.bodyBuf[:size]
	if _, err := io.ReadFull(s.rw, buff); err != nil {
		return nil, err
	}
	return buff, nil
}

func (s *stream) Send(msg interface{}) error {
	val, ok := msg.([]byte)
	if !ok {
		return errors.New("streamProtocol:method Send Parameter type error")
	}

	sendBuf := new(bytes.Buffer)
	sendBuf.Write(s.headBuf)
	sendBuf.Write(val)
	s.headEncode(sendBuf.Bytes(), len(val))
	return nil
}

func (s *stream) Close() error {
	if closer, ok := s.rw.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func init() {
	protocol.Register("stream", &stream{})
}
