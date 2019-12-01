package main

import (
	"encoding/binary"
	"fmt"
	"github.com/zshorz/shadowsockets/ss"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const logCntDelta = 100

var connCnt int
var nextLogConnCnt = logCntDelta

func run(port, password string) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Printf("error listening port %v: %v\n", port, err)
		os.Exit(1)
	}
	passwdManger.add(port, password, ln)
	var cipher *ss.Cipher
	log.Printf("server listening port %v ...\n", port)

	if printURI != "" {
		str := createURI(config.Method, password, printURI, port)
		log.Printf("%s\n", str)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			if debug {
				ss.Debug.Printf("accept error: %v\n", err)
			}
			return
		}
		// 首次连接 创建加密
		if cipher == nil {
			log.Println("creating cipher for port:", port)
			cipher, err = ss.NewCipher(config.Method, password)
			if err != nil {
				log.Printf("Error generating cipher for port: %s %v\n", port, err)
				conn.Close()
				continue
			}
		}
		// 新的加密连接
		go handleConnection(ss.NewConn(conn, cipher.Copy()), port)
	}
}

func handleConnection(conn *ss.Conn, port string) {
	var host string
	connCnt++
	// TODO: 或许不需要记录
	// 累计连接数达到一定数量记录一次，可能不太准
	if connCnt-nextLogConnCnt >= 0 {
		log.Printf("Number of client connections reaches %d\n", nextLogConnCnt)
		nextLogConnCnt += logCntDelta
	}
	if debug {
		ss.Debug.Printf("new client %s -> %s\n", sanitizeAddr(conn.RemoteAddr()), conn.LocalAddr())
	}
	closed := false
	defer func() {
		if debug {
			ss.Debug.Printf("closed pipe %s <-> %s\n", sanitizeAddr(conn.RemoteAddr()), host)
		}
		if !closed {
			conn.Close()
		}
	}()

	host, err := getRequest(conn)
	if err != nil {
		if debug {
			ss.Debug.Println("error getting request", sanitizeAddr(conn.RemoteAddr()), conn.LocalAddr(), err)
		}
		return
	}
	// 保证地址中没有 nil 字符，这在win下会panic
	if strings.ContainsRune(host, 0x00) {
		ss.Debug.Println("invalid domain name.")
		return
	}
	// 开始连接目标
	remote, err := net.Dial("tcp", host)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
			// 文件描述符上限
			ss.Debug.Println("dial error:", err)
		} else {
			ss.Debug.Println("error connecting to:", host, err)
		}
		return
	}
	defer func() {
		if !closed {
			remote.Close()
		}
	}()
	// 开始交换数据
	// TODO: Traffic 需要获取锁，影响效率
	//go func() {
	//	ss.PipeThenClose(conn, remote, func(Traffic int) {
	//		passwdManger.addTraffic(port, Traffic)
	//	})
	//}()
	//
	//ss.PipeThenClose(remote, conn, func(Traffic int) {
	//	passwdManger.addTraffic(port, Traffic)
	//})
	go ss.PipeThenClose(conn, remote, nil)
	ss.PipeThenClose(remote, conn, nil)

	closed = true
	return
}

// getRequest 返回要代理的 host ip/domain:port
// ss 直接的握手 local 把代理目标发过来 | ATYP | DST.ADDR | DST.PORT |
func getRequest(conn *ss.Conn) (host string, err error) {
	ss.SetReadTimeout(conn)
	// 1 + 1 + 256 +2 大于这个数字都行
	buf := make([]byte, 269)
	// 至少读出 ATYP 字段
	if _, err = io.ReadFull(conn, buf[:ss.S5TypeIdx+1]); err != nil {
		return
	}

	// 地址部分开始和结束
	var reqStart, reqEnd int
	addrType := buf[ss.S5TypeIdx]
	switch addrType & ss.AddrMask {
	case ss.S5ipv4:
		reqStart, reqEnd = ss.S5IP0Idx, ss.S5IP0Idx+ss.S5lenIPv4-1 // 减去1 是因为不算 ATYP 字段
	case ss.S5ipv6:
		reqStart, reqEnd = ss.S5IP0Idx, ss.S5IP0Idx+ss.S5lenIPv6-1
	case ss.S5domain:
		// 读出 域名长度
		if _, err = io.ReadFull(conn, buf[ss.S5TypeIdx+1:ss.S5DmLenIdx+1]); err != nil {
			return
		}
		reqStart, reqEnd = ss.S5Dm0Idx, ss.S5Dm0Idx+int(buf[ss.S5DmLenIdx])+ss.S5lenPort // port 长度 2
	default:
		err = fmt.Errorf("addr type %d not supported", addrType&ss.AddrMask)
		return
	}
	if _, err = io.ReadFull(conn, buf[reqStart:reqEnd]); err != nil {
		return
	}
	// req 读取完成

	// typeIP的返回字符串不是最有效的，
	// 但是浏览器（Chrome，Safari，Firefox）似乎都只使用typeDm。 因此，这不是一个大问题。
	switch addrType & ss.AddrMask {
	case ss.S5ipv4:
		host = net.IP(buf[ss.S5IP0Idx : ss.S5IP0Idx+net.IPv4len]).String()
	case ss.S5ipv6:
		host = net.IP(buf[ss.S5IP0Idx : ss.S5IP0Idx+net.IPv6len]).String()
	case ss.S5domain:
		host = string(buf[ss.S5Dm0Idx : ss.S5Dm0Idx+int(buf[ss.S5DmLenIdx])])
	}
	// 解析端口
	port := binary.BigEndian.Uint16(buf[reqEnd-2 : reqEnd])
	host = net.JoinHostPort(host, strconv.Itoa(int(port)))
	return
}
