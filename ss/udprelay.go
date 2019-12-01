package ss

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	reqList            = newReqList()
	natlist            = newNatTable()
	udpTimeout         = 30 * time.Second
	reqListRefreshTime = 5 * time.Minute
)

type natTable struct {
	sync.Mutex
	conns map[string]net.PacketConn
}

func newNatTable() *natTable {
	return &natTable{
		conns: map[string]net.PacketConn{},
	}
}

func (table *natTable) Delete(index string) net.PacketConn {
	table.Lock()
	defer table.Unlock()
	c, ok := table.conns[index]
	if ok {
		delete(table.conns, index)
		return c
	}
	return nil
}

func (table *natTable) Get(index string) (c net.PacketConn, ok bool, err error) {
	table.Lock()
	defer table.Unlock()
	c, ok = table.conns[index]
	if !ok {
		c, err = net.ListenPacket("udp", "")
		if err != nil {
			return nil, false, err
		}
		table.conns[index] = c
	}
	return
}

type requestHeaderList struct {
	sync.Mutex
	List map[string]([]byte)
}

func newReqList() *requestHeaderList {
	ret := &requestHeaderList{
		List: map[string]([]byte){},
	}
	go func() {
		for {
			time.Sleep(reqListRefreshTime)
			ret.Refresh()
		}
	}()
	return ret
}

func (r *requestHeaderList) Refresh() {
	r.Lock()
	defer r.Unlock()
	for k := range r.List {
		delete(r.List, k)
	}
}

func (r *requestHeaderList) Get(dstaddr string) (req []byte, ok bool) {
	r.Lock()
	defer r.Unlock()
	req, ok = r.List[dstaddr]
	return
}

func (r *requestHeaderList) Put(dstaddr string, req []byte) {
	r.Lock()
	defer r.Unlock()
	r.List[dstaddr] = req
	return
}

// 处理地址 返回 socks5 协议头 地址信息
func parseHeaderFromAddr(addr net.Addr) ([]byte, int) {
	// 如果 request addr 类型是域名， 不能反向查找
	ip, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, 0
	}
	buf := make([]byte, 20)
	IP := net.ParseIP(ip)
	b1 := IP.To4()
	iplen := 0
	if b1 == nil { // ipv6
		b1 = IP.To16()
		buf[0] = S5ipv6
		iplen = net.IPv6len
	} else { // ipv4
		buf[0] = S5ipv4
		iplen = net.IPv4len
	}
	copy(buf[1:], b1)
	port_i, _ := strconv.Atoi(port)
	binary.BigEndian.PutUint16(buf[1+iplen:], uint16(port_i))
	return buf[:1+iplen+2], 1 + iplen + 2
}

// 通过 write 将 readClose 发来的数据 发送给 writeaddr 这个地址
func Pipeloop(write net.PacketConn, writeAddr net.Addr, readClose net.PacketConn, addTraffic func(int)) {
	buf := leakyBuf.Get()
	defer leakyBuf.Put(buf)
	defer readClose.Close()
	for {
		readClose.SetDeadline(time.Now().Add(udpTimeout))
		n, raddr, err := readClose.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(*net.OpError); ok {
				if ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE {
					// log too many open file error
					Debug.Println("[udp]read err:", err)
				}
				Debug.Printf("[udp]closed pipe %s<-%s\n", writeAddr, readClose.LocalAddr())
				return
			}
		}
		// TODO: 这里需要改进
		if req, ok := reqList.Get(raddr.String()); ok {
			n, _ := write.WriteTo(append(req, buf[:n]...), writeAddr)
			if addTraffic != nil {
				addTraffic(n)
			}
		} else {
			header, hlen := parseHeaderFromAddr(raddr)
			n, _ := write.WriteTo(append(header[:hlen], buf[:n]...), writeAddr)
			if addTraffic != nil {
				addTraffic(n)
			}
		}
	} // loop
}

// | ATYP | DST.ADDR | DST.PORT |   DATA   |
// handle 是服务器的udp连接， src 是 ss-local 地址， n,receive 是 ss-local 发来的数据
func handleUDPConnection(handle *SecurePacketConn, n int, src net.Addr, receive []byte, addTraffic func(int)) {
	var dstIP net.IP
	var reqLen int
	addrType := receive[S5TypeIdx]
	defer leakyBuf.Put(receive)

	switch addrType & AddrMask {
	case S5ipv4:
		reqLen = S5lenIPv4
		if len(receive) < reqLen {
			Debug.Println("[udp]invalid received message.")
		}
		dstIP = net.IP(receive[S5IP0Idx : S5IP0Idx+net.IPv4len])
	case S5ipv6:
		reqLen = S5lenIPv6
		if len(receive) < reqLen {
			Debug.Println("[udp]invalid received message.")
		}
		dstIP = net.IP(receive[S5IP0Idx : S5IP0Idx+net.IPv6len])
	case S5domain:
		reqLen = int(receive[S5DmLenIdx]) + S5lenDmBase // 域名长度 加上其他头
		if len(receive) < reqLen {
			Debug.Println("[udp]invalid received message.")
		}
		name := string(receive[S5Dm0Idx : S5Dm0Idx+int(receive[S5DmLenIdx])]) // 取出域名
		// 保证域名中没有 nil 字符，否则在win上会 panic
		if strings.ContainsRune(name, 0x00) {
			fmt.Println("[udp]invalid domain name.")
		}
		// dns 解析
		dIP, err := net.ResolveIPAddr("ip", name)
		if err != nil {
			Debug.Printf("[udp]failed to resolve domain name: %s\n", name)
			return
		}
		dstIP = dIP.IP
	default:
		Debug.Printf("[udp]addrType %d not supported", addrType)
		return
	}

	// 地址解析完成， 添加缓存
	dst := &net.UDPAddr{
		IP:   dstIP,
		Port: int(binary.BigEndian.Uint16(receive[reqLen-2 : reqLen])),
		Zone: "",
	}
	if _, ok := reqList.Get(dst.String()); !ok { // 没有则添加
		req := make([]byte, reqLen)
		copy(req, receive)
		reqList.Put(dst.String(), req)
	}
	// 添加到转发表
	//
	remote, exist, err := natlist.Get(src.String())
	if err != nil {
		return
	}
	if !exist {
		Debug.Printf("[udp]new client %s->%s via %s\n", src, dst, remote.LocalAddr()) // 新的连接请求
		go func() {                                                                   // 启动转发 remote -> src
			Pipeloop(handle, src, remote, addTraffic)
			natlist.Delete(src.String())
		}()
	} else { // 老的连接 已经启动了
		Debug.Printf("[udp]using cached client %s->%s via %s\n", src, dst, remote.LocalAddr())
	}
	if remote == nil {
		fmt.Printf("WTF: no remote info in nat list")
	}

	// 发送数据 给 remote
	remote.SetDeadline(time.Now().Add(udpTimeout))
	n, err = remote.WriteTo(receive[reqLen:n], dst)
	if addTraffic != nil {
		addTraffic(n)
	}
	if err != nil {
		if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
			Debug.Println("[udp]write error:", err)
		} else {
			Debug.Println("[udp]error connecting to:", dst, err)
		}
		if conn := natlist.Delete(src.String()); conn != nil {
			conn.Close()
		}
	}
	return
}

func ReadAndHandleUDPReq(c *SecurePacketConn, addTraffic func(int)) (err error) {
	buf := leakyBuf.Get()
	n, src, err := c.ReadFrom(buf[0:])
	if err != nil {
		return err
	}
	go handleUDPConnection(c, n, src, buf, addTraffic)
	return nil
}
