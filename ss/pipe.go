package ss

import (
	"net"
	"time"
)

func SetReadTimeout(c net.Conn) {
	if readTimeout != 0 {
		c.SetReadDeadline(time.Now().Add(readTimeout))
	}
}

// PipeThenClose 从 src 读取数据送到 dst, 完成后关闭 dst
func PipeThenClose(src, dst net.Conn, addTraffic func(int)) {
	defer dst.Close()
	buf := leakyBuf.Get()
	defer leakyBuf.Put(buf)
	for {
		SetReadTimeout(src)
		n, err := src.Read(buf)
		if n > 0 {
			if _, err := dst.Write(buf[0:n]); err != nil {
				Debug.Println("write:", err)
				break
			}
			if addTraffic != nil {
				addTraffic(n) // 记录流量
			}
		}
		if err != nil {
			//if err != io.EOF {
			//	Debug.Println("read:", err)
			//}
			break
		}
	}
	return
}
