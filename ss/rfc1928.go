package ss

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
	S5ipv4		byte = 0x01
	S5domain 	byte = 0x03
	S5ipv6		byte = 0x04
)
