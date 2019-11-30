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
//
// 代理服务器 -> 客户端， 确认信息
//
// +----+--------+
// |VER | METHOD |
// +----+--------+
// | 1  |   1    |
// +----+--------+
//
// METHOD 认证方式:
//o  X'00' NO AUTHENTICATION REQUIRED
//o  X'01' GSSAPI
//o  X'02' USERNAME/PASSWORD
//o  X'03' to X'7F' IANA ASSIGNED
//o  X'80' to X'FE' RESERVED FOR PRIVATE METHODS
//o  X'FF' NO ACCEPTABLE METHODS
//
// 客户端 -> 代理服务器，代理请求 SOCKS request is formed as follows:
//
//+----+-----+-------+------+----------+----------+
//|VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
//+----+-----+-------+------+----------+----------+
//| 1  |  1  | X'00' |  1   | Variable |    2     |
//+----+-----+-------+------+----------+----------+
//
// Replies
//
// +----+-----+-------+------+----------+----------+
// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
// +----+-----+-------+------+----------+----------+
// | 1  |  1  | X'00' |  1   | Variable |    2     |
// +----+-----+-------+------+----------+----------+
const (
	// 地址类型
	S5ipv4   byte = 0x01
	S5domain byte = 0x03
	S5ipv6   byte = 0x04

	// VER NMETHODS CMD位置，
	S5VerIdx     = 0
	S5NmethodIdx = 1
	S5CmdIdx     = 1

	// ATYP 字段前面的字节数
	S5reqBaseLen = 3
	S5repBaseLen = 3
	// 从 ATYP 字段算起的下标
	S5TypeIdx   = 0                   // address type index
	S5IP0Idx    = 1                   // ip address start index
	S5DmLenIdx  = 1                   // domain address length index
	S5Dm0Idx    = 2                   // domain address start index
	S5lenIPv4   = 1 + net.IPv4len + 2 // 1addrType + ipv4 + 2port
	S5lenIPv6   = 1 + net.IPv6len + 2 // 1addrType + ipv6 + 2port
	S5lenDmBase = 1 + 1 + 2           // 1addrType + 1addrLen + 2port, plus addrLen
	// lenHmacSha1 = 10

	// 认证方式
	S5NoAuthentication byte = 0x00
	// 版本
	S5Ver byte = 0x05
	// CMD 字段
	S5CmdConnect byte = 0x01
	S5CmdBind    byte = 0x02
	S5CmdUdp     byte = 0x03
)

const (
	S5socksVer5       = 5
	S5socksCmdConnect = 1
)
