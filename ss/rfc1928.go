package ss

import "net"

// 客户端 -> 代理服务器， 请求认证:
//
// +----+----------+----------+
// |VER | NMETHODS | METHODS  |
// +----+----------+----------+
// | 1  |    1     | 1 to 255 |
// +----+----------+----------+
//
// VER 版本号 固定为 5
// NMETHODS 可供选择的认证方法 选了多少种
// METHODS 选择的方法
const (
	S5ipv4   byte = 0x01
	S5domain byte = 0x03
	S5ipv6   byte = 0x04

	// udp
	S5TypeIdx   = 0                   // address type index
	S5IP0Idx    = 1                   // ip address start index
	S5DmLenIdx  = 1                   // domain address length index
	S5Dm0Idx    = 2                   // domain address start index
	S5lenIPv4   = 1 + net.IPv4len + 2 // 1addrType + ipv4 + 2port
	S5lenIPv6   = 1 + net.IPv6len + 2 // 1addrType + ipv6 + 2port
	S5lenDmBase = 1 + 1 + 2           // 1addrType + 1addrLen + 2port, plus addrLen
	// lenHmacSha1 = 10
)

const (
	S5socksVer5       = 5
	S5socksCmdConnect = 1
)
