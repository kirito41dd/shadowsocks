package ss

// buffer pool 重复使用避免申请内存的开销
type LeakyBuf struct {
	bufSize  int // size of each buffer
	freeList chan []byte
}

const leakyBufSize = 5120 // bufsize 设置为 5KB， 但这对udp可能不够
// udp 包最大长度 为 65535，但绝大部分udp都将长度控制在1472byte以下（可以避免ip层分片）
const maxNBuf = 2048

// 负责分发和回收内部使用的 buffer， 重复使用避免申请内存的开销
var leakyBuf = NewLeakyBuf(maxNBuf, leakyBufSize)

// NewLeakyBuf 创建一个 leaky buffer, 可以包含 n 个 buffer， 每个大小为 bufSize
func NewLeakyBuf(n, bufSize int) *LeakyBuf {
	return &LeakyBuf{
		bufSize:  bufSize,
		freeList: make(chan []byte, n),
	}
}

// Get 从 leaky buffer 中返回一个 buffer， 或者创建一个新的 buffer
func (lb *LeakyBuf) Get() (b []byte) {
	select {
	case b = <-lb.freeList:
	default:
		b = make([]byte, lb.bufSize)
	}
	return
}

// Put 在 free buffer pool 中加入一个 buffer， 如果 buffer 的大小和 leaky buffer中的不一致
// 将引发一个 panic， 以此来暴漏错误的用法
func (lb *LeakyBuf) Put(b []byte) {
	if len(b) != lb.bufSize {
		panic("invalid buffer size that's put into leaky buffer")
	}
	select {
	case lb.freeList <- b:
	default:
	}
	return
}
