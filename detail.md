## shadowsocks共工作原理

```txt
                                    |
                        client <----|----> remote                    (×)
                                    |
                                    |
          (socks5)               (shadow) 
client  <---------->  ss-local <----|----> ss-server <--> remote     (√)
                                    |
                                    |
                                   GFW
```

其中：

* client 是浏览器或其他需要代理的应用
* remote 是像谷歌这样无法直接访问的网站
* client 和 ss-local 之间的握手协议是 socks5 , rfc1928
* ss-local 和 ss-server 之间使用加密信道以及简单的握手
* GFW 是防火长城



## socks5握手细节

详细见 rfc1928，以下只说明tcp的代理方式（udp很简单）

* client to ss-local 请求认证

```txt
 +----+----------+----------+
 |VER | NMETHODS | METHODS  |
 +----+----------+----------+
 | 1  |    1     | 1 to 255 |
 +----+----------+----------+
// VER 版本号 固定为 5
// NMETHODS 可供选择的认证方法 选了多少种
// METHODS 选择的方法
一个可能的数据 byte: 0x05 0x01 0x00
```

* ss-local to client 确认认证方式

```txt
 +----+--------+
 |VER | METHOD |
 +----+--------+
 | 1  |   1    |
 +----+--------+
// METHOD 认证方式
一个无需认证的回答 byte: 0x05 0x00
```

* client to ss-local 代理请求
```text
 +----+-----+-------+------+----------+----------+
 |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
 +----+-----+-------+------+----------+----------+
 | 1  |  1  | X'00' |  1   | Variable |    2     |
 +----+-----+-------+------+----------+----------+
// CMD 代理类型 tcp bind or udp
// RSV 保留字段
// ATYP 地址类型 ipv4 ipv6 domain
// DST.ADDR 如果地址是ip，就是ip的二进制形式，如果是域名，第一个字节为域名长度
// DST.PORT 端口
比如想要以tcp方式代理 google.com 端口 0
byte: 0x05 0x01 0x00 0x03 0x0A b`google.com` 0x00 0x00
```

   * ss-local to client 回应 

```text
 +----+-----+-------+------+----------+----------+
 |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
 +----+-----+-------+------+----------+----------+
 | 1  |  1  | X'00' |  1   | Variable |    2     |
 +----+-----+-------+------+----------+----------+
// REP 0x00 表示成功
// ATYP  BND.ADDR  BND.PORT 和请求一样，描述了一个地址(大多时候并不重要)
成功的答复, 随便ipv4返回了 0.0.0.0:0
byte: 0x05 0x00 0x00 0x01 0x00 0x00 x00 0x00 0x00 0x00
```

至此socks5已经完成了，双方开始交换数据，ss-local 发送给 client 的数据需要从 ss-server 取得



## ss-local 和 ss-server 的握手

它们之间通过事先商议好的加密方式建立加密连接

* 建立加密连接的过程

使用对称加密，在tcp连接建立后双方互换`iv` (对称加密使用同一个key, 但是使用不同的iv)，双方都有自己的`iv`加密数据，所以事先要交换，交换过后加密信道建立完成

下面交换的数据都经过加密

* ss-local to ss-server 代理请求

```text
 +------+----------+----------+
 | ATYP | DST.ADDR | DST.PORT |
 +------+----------+----------+
 |  1   | Variable |    2     |
 +------+----------+----------+
// 就是 client 发给 ss-local 的地址部分
以tcp方式代理 google.com 端口 0
byte: 0x03 0x0A b`google.com` 0x00 0x00
```

至此 ss-local 和 ss-server 握手完成

ss-server 连接 remote ，开始交换数据

## 数据流动

ss-local 无脑转发 client 的数据给 ss-server

ss-server 无脑转发给 remote

remote 的回复被 ss-server 送给 ss-local

ss-local 无脑转发 ss-server 的回复给 client



* 更多细节请在代码中寻找