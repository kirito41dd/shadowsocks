package ss

// 加密 解密

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"encoding/binary"
	"errors"
	"golang.org/x/crypto/salsa20/salsa"
	"io"

	"github.com/aead/chacha20"
	"golang.org/x/crypto/blowfish"
	"golang.org/x/crypto/cast5"
)

// 定义一个错误，密码为空
var errEmptyPassword = errors.New("empty key")

func md5sum(d []byte) []byte {
	h := md5.New()
	h.Write(d)
	return h.Sum(nil)
}

func evpBytesToKey(password string, keyLen int) (key []byte) {
	const md5Len = 16

	// cnt 是 md5Len 的整倍数
	cnt := (keyLen - 1) / md5Len + 1
	m := make([]byte, cnt * md5Len)
	copy(m, md5sum([]byte(password)))

	// 重复调用md5，直到生成的字节足够为止。
	// 每次调用md5都会使用数据 prev md5 sum + password
	d := make([]byte, md5Len + len(password))
	start := 0
	for i := 1; i < cnt; i++ {
		start += md5Len
		copy(d, m[start - md5Len : start])
		copy(d[md5Len:], password)
		copy(m[start:], md5sum(d))
	}
	return m[:keyLen]
}

type DecOrEnc int

// 常量代表 加密 解密
const (
	Decrypt DecOrEnc = iota
	Encrypt
)

// 各种加密协议的 stream 初始化函数

func newStream(block cipher.Block, err error, key, iv []byte, doe DecOrEnc) (cipher.Stream, error) {
	if err != nil {
		return nil, err
	}
	if doe == Encrypt {
		return cipher.NewCFBEncrypter(block, iv), nil
	} else {
		return cipher.NewCFBDecrypter(block, iv), nil
	}
}

func newAESCFBStream(key, iv []byte, doe DecOrEnc) (cipher.Stream, error) {
	block, err := aes.NewCipher(key)
	return newStream(block, err, key, iv, doe)
}

func newAESCTRStream(key, iv []byte, doe DecOrEnc) (cipher.Stream, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewCTR(block, iv), nil
}

func newDESStream(key, iv []byte, doe DecOrEnc) (cipher.Stream, error) {
	block, err := des.NewCipher(key)
	return newStream(block, err, key, iv, doe)
}

func newBlowFishStream(key, iv []byte, doe DecOrEnc) (cipher.Stream, error) {
	block, err := blowfish.NewCipher(key)
	return newStream(block, err, key, iv, doe)
}

func newCast5Stream(key, iv []byte, doe DecOrEnc) (cipher.Stream, error) {
	block, err := cast5.NewCipher(key)
	return newStream(block, err, key, iv, doe)
}

func newRC4MD5Stream(key, iv []byte, _ DecOrEnc) (cipher.Stream, error) {
	h := md5.New()
	h.Write(key)
	h.Write(iv)
	rc4key := h.Sum(nil)

	return rc4.NewCipher(rc4key)
}

func newChaCha20Stream(key, iv []byte, _ DecOrEnc) (cipher.Stream, error) {
	return chacha20.NewCipher(iv, key)
}

func newChaCha20IETFStream(key, iv []byte, _ DecOrEnc) (cipher.Stream, error) {
	return chacha20.NewCipher(iv, key)
}

type salsaStreamCipher struct {
	nonce 	[8]byte
	key 	[32]byte
	counter	int
}

func (c *salsaStreamCipher) XORKeyStream(dst, src []byte) {
	var buf []byte
	padLen := c.counter % 64
	dataSize := len(src) + padLen
	if cap(dst) >= dataSize {
		buf = dst[:dataSize]
	} else if leakyBufSize >= dataSize {
		buf = leakyBuf.Get()
		defer leakyBuf.Put(buf)
		buf = buf[:dataSize]
	} else {
		buf = make([]byte, dataSize)
	}

	var subNonce [16]byte
	copy(subNonce[:], c.nonce[:])
	binary.LittleEndian.PutUint64(subNonce[len(c.nonce):], uint64(c.counter/64))

	// It's difficult to avoid data copy here. src or dst maybe slice from
	// Conn.Read/Write, which can't have padding.
	copy(buf[padLen:], src[:])
	salsa.XORKeyStream(buf, buf, &subNonce, &c.key)
	copy(dst, buf[padLen:])

	c.counter += len(src)
}

func newSalsa20Stream(key, iv []byte, _ DecOrEnc) (cipher.Stream, error) {
	var c salsaStreamCipher
	copy(c.nonce[:], iv[:8])
	copy(c.key[:], key[:32])
	return &c, nil
}


// 保存加密协议的必要信息
type cipherInfo struct {
	keyLen 	int
	ivLen	int
	newStream func(key, iv []byte, doe DecOrEnc) (cipher.Stream, error)
}

// 支持的所有加密协议
var cipherMethod = map[string]*cipherInfo{
	"aes-128-cfb":		{16, 16, newAESCFBStream},
	"aes-192-cfb":   	{24, 16, newAESCFBStream},
	"aes-256-cfb":   	{32, 16, newAESCFBStream},
	"aes-128-ctr":   	{16, 16, newAESCTRStream},
	"aes-192-ctr":   	{24, 16, newAESCTRStream},
	"aes-256-ctr":   	{32, 16, newAESCTRStream},
	"des-cfb":      	{8, 8, newDESStream},
	"bf-cfb":       	{16, 8, newBlowFishStream},
	"cast5-cfb":    	{16, 8, newCast5Stream},
	"rc4-md5":       	{16, 16, newRC4MD5Stream},
	"rc4-md5-6":     	{16, 6, newRC4MD5Stream},
	"chacha20":      	{32, 8, newChaCha20Stream},
	"chacha20-ietf": 	{32, 12, newChaCha20IETFStream},
	"salsa20":       	{32, 8, newSalsa20Stream},
}

func CheckCipherMethod(method string) error {
	if method == "" {
		method = "aes-256-cfb"
	}
	_, ok := cipherMethod[method]
	if !ok {
		return errors.New("Unsupported encryption method: " + method)
	}
	return nil
}

// 负责提供加密解密功能
type Cipher struct {
	enc 	cipher.Stream
	dec 	cipher.Stream
	key 	[]byte
	info 	*cipherInfo
}

// NewCipher 创建一个 Cipher
// 使用 cipher.Copy() 可以创建一个使用相同加密算法和密码的 cipher
func NewCipher(method, password string) (c *Cipher, err error) {
	if password == "" {
		return nil, errEmptyPassword
	}
	mi, ok := cipherMethod[method]
	if !ok {
		return  nil, errors.New("Unsupported encryption method: " + method)
	}

	key := evpBytesToKey(password, mi.keyLen)

	c = &Cipher{key: key, info: mi}

	return c, nil
}

// 初始化加密 stream ， 返回 iv
func (c *Cipher) initEncrypt() (iv []byte, err error) {
	iv = make([]byte, c.info.ivLen)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	c.enc, err = c.info.newStream(c.key, iv, Encrypt)
	return
}

func (c *Cipher) initDecrypt(iv []byte) (err error) {
	c.dec, err = c.info.newStream(c.key, iv, Decrypt)
	return
}

func (c *Cipher) encrypt(dst, src []byte) {
	c.enc.XORKeyStream(dst, src)
}

func (c *Cipher) decrypt(dst, src []byte) {
	c.dec.XORKeyStream(dst, src)
}

// Copy 返回一个使用相同加密算法和密码的 cipher
// enc 和 dec 都为 nil
func (c *Cipher) Copy() *Cipher {
	nc := *c
	nc.enc = nil
	nc.dec = nil
	return &nc
}







