package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"io"
	"sync"
)

type Protocol interface {
	Register(interface{})
	NewCodec(io.ReadWriter) (Codec, error)
}

type Codec interface {
	Receive() (interface{}, error)
	Send(interface{}) error
	Close() error
}

var (
	procotolMux sync.RWMutex
	adapters = make(map[string]Protocol)
)

func Register(name string, adpater Protocol) {
	procotolMux.Lock()
	defer procotolMux.Unlock()
	if adpater == nil {
		panic("Protocol:adapter is nil")
	}

	if _, ok := adapters[name]; ok {
		panic("Protocol:Register called twice for adapter" + name)
	}
	adapters[name] = adpater
}

//实例化协议，根据config来决定是否使用bufio或者字节长度来解码
//使用bufio示例： rotocol.NewProtocol("json",{"bufio":{"readSize":"1024","writeSize":"1024"}}) bufio 可配置项：readSize, writeSize  如果不配置，则使用默认值
//使用字节长度解析示例：rotocol.NewProtocol("json",{"fixlen":{}}) fixlen 可配置maxSend,maxRecv,n byteOrder  如果不配置，则使用默认值
//使用bufio+fixlen示例：`{"fixlen":{},"bufio":{}}`
//都不使用 NewProtocol("json","")
func NewProtocol(name string, config string) (Protocol, error) {
	adapter, ok := adapters[name]
	if !ok {
		err := fmt.Errorf("Protocol: unknown adapter name %q (forgot to import?)", name)
		return nil, err
	}

	var cfg map[string]interface{}
	json.Unmarshal([]byte(config),&cfg)

	if fixlen,ok := cfg["fixlen"]; ok {
		fixMap := fixlen.(map[string]interface{})
		bs,_ := json.Marshal(fixMap)

		var err error
		if adapter, err = NewFixlenProtocol(string(bs),adapter); err != nil {
			return nil,err
		}
	}

	if bio,ok := cfg["bufio"]; ok {
		bioMap := bio.(map[string]interface{})
		bs,_ := json.Marshal(bioMap)

		var err error
		if adapter, err = NewBufioProtocol(string(bs),adapter); err != nil {
			return nil,err
		}
	}
	return adapter,nil
}

//实例化字节长度解析器协议
func NewFixlenProtocol(config string, base Protocol) (Protocol,error) {
	cfg := make(map[string]string)
	json.Unmarshal([]byte(config), &cfg)

	if _,ok := cfg["n"]; !ok {
		cfg["n"] = strconv.Itoa(defaultPackHead)
	}
	if _,ok := cfg["maxSend"]; !ok {
		cfg["maxSend"] = strconv.Itoa(0)
	}
	if _,ok := cfg["maxRecv"]; !ok {
		cfg["maxRecv"] = strconv.Itoa(0)
	}

	n,_ := strconv.Atoi(cfg["n"])
	configNMap := map[int]int{1:1, 2:1, 4:1, 8:1}
	if _,ok := configNMap[n]; !ok {
		return nil, errors.New("streamProtocol: n is invalid ")
	}

	var byteOrder binary.ByteOrder
	if _,ok := cfg["byteOrder"]; !ok {
		byteOrder = defaultByteOrder
	}else {
		if cfg["byteOrder"] == "littleEndian" {
			byteOrder = binary.LittleEndian
		}else{
			byteOrder = binary.BigEndian
		}
	}
	maxSend,_ := strconv.Atoi(cfg["maxSend"])
	maxRecv,_ := strconv.Atoi(cfg["maxRecv"])
	fixlenProtocol, err := newFixlenProtocol(base, maxSend, maxRecv, n, byteOrder)
	if err != nil {
		return nil, err
	}
	return fixlenProtocol, nil
}

//实例化bufio协议解析器
func NewBufioProtocol(config string, base Protocol) (Protocol,error) {
	cfg := make(map[string]string)
	json.Unmarshal([]byte(config), &cfg)

	if _,ok := cfg["readSize"]; !ok {
		cfg["readSize"] = strconv.Itoa(0)
	}
	if _,ok := cfg["writeSize"]; !ok {
		cfg["writeSize"] = strconv.Itoa(0)
	}

	readSize,_ := strconv.Atoi(cfg["readSize"])
	writeSize,_ := strconv.Atoi(cfg["writeSize"])

	bioProtocol,_ := newBufio(base , readSize, writeSize)
	return bioProtocol, nil
}
