package protobuf

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/gary163/seals/protocol"
	"github.com/golang/protobuf/proto"
)

type protobufProtocol struct {
	types map[string]reflect.Type
	names map[reflect.Type]string
}

func (j *protobufProtocol) Register(t interface{}) {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		fmt.Printf("Register relect.prt elem:%v\n",rt)
	}
	path := rt.PkgPath() + "/" + rt.Name()
	bys := strings.Split(path,"/")
	l := len(bys)
	prefix := ""
	if l-2 >= 0 {
		prefix = bys[l-2]
	}
	name := fmt.Sprintf("%v_%v",prefix,bys[l-1])
	j.types[name] = rt
	j.names[rt] = name
	fmt.Printf("types:%+v\n",j.types)
	fmt.Printf("names:%+v\n",j.names)
}

func (p *protobufProtocol) NewCodec(rw io.ReadWriter) (protocol.Codec, error) {
	codec := &protobufCodec{
		p  :  p,
		rw : rw,
	}
	codec.closer, _ = rw.(io.Closer)
	return codec, nil
}

type in struct {
	Head  *string   `protobuf:"bytes,1,opt,name=head,proto3" json:"head,omitempty"`
	Body  []byte
}

func (i *in) Reset()         { *i = in{} }
func (i *in) String() string { return proto.CompactTextString(i) }
func (*in) ProtoMessage()    {}

type out struct {
	Head *string `protobuf:"bytes,1,opt,name=head,proto3" json:"head,omitempty"`
	Body proto.Message
}

func (o *out) Reset()         { *o = out{} }
func (o *out) String() string { return proto.CompactTextString(o) }
func (*out) ProtoMessage()    {}

type protobufCodec struct {
	p  *protobufProtocol
	rw io.ReadWriter
	closer io.Closer
}

func (c *protobufCodec) Receive() (interface{}, error) {
	recv,err := ioutil.ReadAll(c.rw)
	fmt.Printf("Receive:%v\n",recv)
	if err != nil {
		return nil,err
	}
	var in in
	if err := proto.Unmarshal(recv,&in); err != nil {
		return nil,err
	}

	var body interface{}
	if in.Head != nil {
		name := *in.Head
		if t, exists := c.p.types[name]; exists {
			body = reflect.New(t).Interface()
		}
	}
	err = proto.Unmarshal(in.Body, body.(proto.Message))
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *protobufCodec) Send(msg interface{}) error {
	var out out
	t := reflect.TypeOf(msg)
	if t.Kind() == reflect.Ptr {

		t = t.Elem()
		fmt.Printf("send reflect.ptr :%v\n",t)
	}
	fmt.Printf("t:%v\n",t)
	if name, exists := c.p.names[t]; exists {
		out.Head = proto.String(name)
	}
	fmt.Printf("name:%v\n",out.Head)
	out.Body = msg.(proto.Message)
	fmt.Println(out.Body )

	outBytes,err := proto.Marshal(&out)
	if err != nil {
		fmt.Printf("Marshal err:%v\n",err)
		return err
	}
	fmt.Printf("out:%+v,outbytes:%v\n",out,outBytes)
	fmt.Printf("string(outBytes):%s\n",string(outBytes))
	_,err = c.rw.Write(outBytes)
	return err
}

func (c *protobufCodec) Close() error {
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}

func init() {
	protocol.Register("protobuf", &protobufProtocol{
		types: make(map[string]reflect.Type),
		names: make(map[reflect.Type]string),
	})
}
