package ss

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

const (
	AddrMask byte = 0xf
)

type Conn struct {
	net.Conn			// 内嵌接口，可以使用所有实现了 net.Conn 接口的实例来初始化
	*Cipher				// 组合，Conn 可以直接使用 Cipher 成员  [ˈsaɪfər] 密码; 暗号;
	readBuf 	[]byte
	writeBuf 	[]byte
}

func NewConn(c net.Conn, cipher *Cipher) *Conn {
	return &Conn{
		Conn:     c,
		Cipher:   cipher,
		readBuf:  leakyBuf.Get(),
		writeBuf: leakyBuf.Get(),
	}
}

func (c *Conn) Close() error {
	leakyBuf.Put(c.readBuf)
	leakyBuf.Put(c.writeBuf)
	return c.Conn.Close()
}

// 处理原始地址，返回 sock5 协议中地址信息, 从ATYP字段开始。
func RawAddr(addr string) (buf []byte, err error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("shadowsocks: address error %s %v", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("shadowsocks: invalid port %s", addr)
	}

	hostLen := len(host)
	l := 1 + 1 + hostLen + 2 // addrType + lenByte + address + port
	buf = make([]byte, l)
	buf[0] = S5domain		// 3 means the address is domain name
	buf[1] = byte(hostLen)	// host address length  followed by host address
	copy(buf[2:], host)
	binary.BigEndian.PutUint16(buf[2+hostLen:2+hostLen+2], uint16(port))
	return
}

// DialWithRawAddr 为实现本地SOCKS代理的用户使用
// rawaddr应该包含socks请求中的部分数据，从ATYP字段开始。
func DialWithRawAddr(rawaddr []byte, server string, cipher *Cipher) (c *Conn, err error) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return
	}
	c = NewConn(conn, cipher)
	if _, err = c.Write(rawaddr); err != nil {
		c.Close()
		return nil, err
	}
	return
}

// Dial 地址 addr 应该是 host:port 的形式
func Dial(addr, server string, cipher *Cipher) (c *Conn, err error) {
	ra, err := RawAddr(addr)
	if err != nil {
		return
	}
	return DialWithRawAddr(ra, server, cipher)
}

func (c *Conn) Read(b []byte) (n int, err error) {
	// 只有连接刚开始才会发送 iv 信息，之后的的包不携带 iv
	if c.dec != nil {
		iv := make([]byte, c.info.ivLen)
		if _, err = io.ReadFull(c.Conn, iv); err != nil {
			return
		}
		// TODO: go on
	}
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	// 只有连接刚开始才会发送 iv 信息，之后的的包不携带 iv
	//var iv []byte
	if c.enc == nil {

		// TODO: go on
	}
	return
}


