package ss

// ss client 与 ss server 之间的连接

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

const (
	AddrMask byte = 0xf
)

type Conn struct {
	net.Conn // 内嵌接口，可以使用所有实现了 net.Conn 接口的实例来初始化
	*Cipher  // 组合，Conn 可以直接使用 Cipher 成员  [ˈsaɪfər] 密码; 暗号;
	readBuf  []byte
	writeBuf []byte
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
	buf[0] = S5domain      // 3 means the address is domain name
	buf[1] = byte(hostLen) // host address length  followed by host address
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
	// ss client server 两端 互换 iv
	if c.dec == nil {
		iv := make([]byte, c.info.ivLen)
		if _, err = io.ReadFull(c.Conn, iv); err != nil {
			return
		}
		if err = c.initDecrypt(iv); err != nil {
			return
		}
	}

	cipherData := c.readBuf
	if len(b) > len(cipherData) { // 大小不够
		cipherData = make([]byte, len(b))
	} else {
		cipherData = cipherData[:len(b)]
	}

	n, err = c.Conn.Read(cipherData)
	if n > 0 {
		c.decrypt(b[0:n], cipherData[0:n])
	}
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	// 只有连接刚开始才会发送 iv 信息，之后的的包不携带 iv
	var iv []byte // nil
	if c.enc == nil {
		iv, err = c.initEncrypt()
		if err != nil {
			return
		}
	}

	cipherData := c.writeBuf
	dataSize := len(b) + len(iv)
	if dataSize > len(cipherData) {
		cipherData = make([]byte, dataSize)
	} else {
		cipherData = cipherData[:dataSize]
	}

	if iv != nil {
		// 说明这是第一次发送信息， 要把 iv 发过去
		copy(cipherData, iv)
	}
	c.encrypt(cipherData[len(iv):], b)
	n, err = c.Conn.Write(cipherData)
	// 这里的 n 是 连着 iv 的长度， 不应该直接返回
	if n >= len(iv) {
		n = n - len(iv)
		return
	} else {
		// FIXME: 错误如何处理 目前是 返回0  重置 enc 此时连接已经不可用了， 不过这种错误概率极小
		n = 0
		err = errors.New("can not write iv")
		c.enc = nil
		return
	}
}
