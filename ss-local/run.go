package main

import (
	"encoding/binary"
	"github.com/zshorz/shadowsocks/ss"
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
)

func run(listenAddr string) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("starting local socks5 server at %v ...\n", listenAddr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	ss.Debug.Printf("socks connect from %s\n", conn.RemoteAddr().String())
	closed := false
	defer func() {
		if !closed {
			conn.Close()
		}
	}()

	var err error = nil
	if err = handShake(conn); err != nil {
		log.Println("socks handshake:", err)
		return
	}
	rawaddr, addr, err := getRequest(conn)
	if err != nil {
		log.Println("error getting request:", err)
		return
	}
	// 发送回复 TODO: 这里目前写死了，应该返回的是欲连接主机的 地址和端口，但这些信息对客户端没啥用
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	//   5   success  0      ipv4   0.0.0.0    1080   网络序
	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	if err != nil {
		ss.Debug.Println("send connection confirmation:", err)
		return
	}

	remote, err := createServerConn(rawaddr, addr)
	if err != nil {
		if len(servers.srvCipher) > 1 {
			log.Println("Failed connect to all available shadowsocks server")
		}
		return
	}
	defer func() {
		if !closed {
			remote.Close()
		}
	}()

	go ss.PipeThenClose(conn, remote, Traffic)
	ss.PipeThenClose(remote, conn, nil)
	closed = true
	ss.Debug.Println("closed connection to", addr)
}

func handShake(conn net.Conn) (err error) {
	// 1 + 1 + 256 详见 Socks5 握手
	buf := make([]byte, 258)

	var n int
	ss.SetReadTimeout(conn)
	// 保证一定收到了 nmethod 字段
	if n, err = io.ReadAtLeast(conn, buf, ss.S5NmethodIdx+1); err != nil {
		return
	}
	if buf[ss.S5VerIdx] != ss.S5Ver {
		return errVer
	}
	nmethod := int(buf[ss.S5NmethodIdx])
	msgLen := nmethod + 2
	if n == msgLen {
		// handshake done
	} else if n < msgLen { // METHODS 字段没有读完
		if _, err = io.ReadFull(conn, buf[n:msgLen]); err != nil {
			return
		} else { // n > msgLen ,  有多余的数据
			return errAuthExtraData
		}
	}
	// 发送确认， version 5, 无需身份验证
	_, err = conn.Write([]byte{ss.S5Ver, ss.S5NoAuthentication})
	return
}

func getRequest(conn net.Conn) (rawaddr []byte, host string, err error) {

	buf := make([]byte, 263)
	var n int
	ss.SetReadTimeout(conn)
	// read 最少要读取出 ATYP 字段多一个字节(doman len)
	if n, err = io.ReadAtLeast(conn, buf, ss.S5reqBaseLen+ss.S5DmLenIdx+1); err != nil {
		return
	}
	// check version and cmd
	if buf[ss.S5VerIdx] != ss.S5Ver {
		err = errVer
		return
	}
	// FIXME: 目前不支持udp
	if buf[ss.S5CmdIdx] != ss.S5CmdConnect {
		err = errCmd
		return
	}

	reqLen := -1
	switch buf[ss.S5reqBaseLen+ss.S5TypeIdx] {
	case ss.S5ipv4:
		reqLen = ss.S5reqBaseLen + ss.S5lenIPv4
	case ss.S5ipv6:
		reqLen = ss.S5reqBaseLen + ss.S5lenIPv6
	case ss.S5domain:
		reqLen = int(buf[ss.S5reqBaseLen+ss.S5DmLenIdx]) + ss.S5lenDmBase + ss.S5reqBaseLen
	default:
		err = errAddrType
		return
	}
	if n == reqLen {

	} else if n < reqLen {
		if _, err = io.ReadFull(conn, buf[n:reqLen]); err != nil {
			return
		}
	} else {
		err = errReqExtraData
		return
	}
	rawaddr = buf[ss.S5reqBaseLen+ss.S5TypeIdx : reqLen]

	switch buf[ss.S5reqBaseLen+ss.S5TypeIdx] {
	case ss.S5ipv4:
		host = net.IP(buf[ss.S5reqBaseLen+ss.S5IP0Idx : ss.S5reqBaseLen+ss.S5IP0Idx+net.IPv4len]).String()
	case ss.S5ipv6:
		host = net.IP(buf[ss.S5reqBaseLen+ss.S5IP0Idx : ss.S5reqBaseLen+ss.S5IP0Idx+net.IPv6len]).String()
	case ss.S5domain:
		host = string(buf[ss.S5reqBaseLen+ss.S5Dm0Idx : ss.S5reqBaseLen+ss.S5Dm0Idx+buf[ss.S5reqBaseLen+ss.S5DmLenIdx]])
	}
	port := binary.BigEndian.Uint16(buf[reqLen-2 : reqLen])
	host = net.JoinHostPort(host, strconv.Itoa(int(port)))

	ss.Debug.Printf("%s request to conn %s\n", conn.RemoteAddr().String(), host)

	return
}

func createServerConn(rawaddr []byte, addr string) (remote *ss.Conn, err error) {
	const baseFailCnt = 20
	n := len(servers.srvCipher)
	skipped := make([]int, 0)
	for i := 0; i < n; i++ {
		if servers.failCnt[i] > 0 && rand.Intn(servers.failCnt[i]+baseFailCnt) != 0 {
			skipped = append(skipped, i)
			continue
		}
		remote, err = connectToServer(i, rawaddr, addr)
		if err == nil {
			return
		}
	}
	//
	for _, i := range skipped {
		remote, err = connectToServer(i, rawaddr, addr)
		if err == nil {
			return
		}
	}
	return nil, err
}

func connectToServer(serverId int, rawaddr []byte, addr string) (remote *ss.Conn, err error) {
	se := servers.srvCipher[serverId]
	remote, err = ss.DialWithRawAddr(rawaddr, se.server, se.cipher.Copy())
	if err != nil {
		ss.Debug.Printf("error connecting to shadowsocks server id %n : %s\n", serverId, err)
		const maxFailCnt = 30
		if servers.failCnt[serverId] < maxFailCnt {
			servers.failCnt[serverId]++
		}
		return nil, err
	}
	ss.Debug.Printf("connected to %s via %s\n", addr, se.server)
	servers.failCnt[serverId] = 0
	return
}
