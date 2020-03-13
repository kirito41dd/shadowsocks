package ss

import (
	"reflect"
	"testing"
)

const text = "You can you up, no can no bb."

// 是否打印加密后的输出
var debug = true

// 检测加密算法正确性

func testCipher(t *testing.T, c *Cipher, msg string) {
	n := len(text)
	cipherBuf := make([]byte, n)
	originTxt := make([]byte, n)

	c.encrypt(cipherBuf, []byte(text)) // 加密
	c.decrypt(originTxt, cipherBuf)    // 解密

	if debug {
		t.Logf("%s encrypt -> %x decrypt -> %s\n", text, cipherBuf, originTxt)
	}
	if string(originTxt) != text {
		t.Error(msg, "encrypt then decrytp does not get original text")
	}
}

func Test_EvpBytesToKey(t *testing.T) {
	key := evpBytesToKey("foobar", 32)
	keyTarget := []byte{0x38, 0x58, 0xf6, 0x22, 0x30, 0xac, 0x3c, 0x91,
		0x5f, 0x30, 0x0c, 0x66, 0x43, 0x12, 0xc6, 0x3f,
		0x56, 0x83, 0x78, 0x52, 0x96, 0x14, 0xd2, 0x2d,
		0xdb, 0x49, 0x23, 0x7d, 0x2f, 0x60, 0xbf, 0xdf}
	if !reflect.DeepEqual(key, keyTarget) {
		t.Errorf("key not correct\n\texpect: %v\n\tgot:   %v\n", keyTarget, key)
	}
}

func testBlockCipher(t *testing.T, method string) {
	var cipher *Cipher
	var err error

	cipher, err = NewCipher(method, "foobar")
	if err != nil {
		t.Fatal(method, "NewCipher:", err)
	}
	cipherCopy := cipher.Copy()

	iv, err := cipher.initEncrypt()
	if err != nil {
		t.Error(method, "initEncrypt:", err)
	}
	if err = cipher.initDecrypt(iv); err != nil {
		t.Error(method, "initDecrypt:", err)
	}

	iv, err = cipherCopy.initEncrypt()
	if err != nil {
		t.Error(method, "copy initEncrypt:", err)
	}
	if err = cipherCopy.initDecrypt(iv); err != nil {
		t.Error(method, "copy initDecrypt:", err)
	}

	testCipher(t, cipherCopy, method+" copy")
}

func Test_AES123CFB(t *testing.T) {
	testBlockCipher(t, "aes-128-cfb")
}

func Test_AES192CFB(t *testing.T) {
	testBlockCipher(t, "aes-192-cfb")
}

func Test_AES256CFB(t *testing.T) {
	testBlockCipher(t, "aes-256-cfb")
}

func Test_AES128CTR(t *testing.T) {
	testBlockCipher(t, "aes-128-ctr")
}

func Test_AES192CTR(t *testing.T) {
	testBlockCipher(t, "aes-192-ctr")
}

func Test_AES256CTR(t *testing.T) {
	testBlockCipher(t, "aes-256-ctr")
}

func Test_DES(t *testing.T) {
	testBlockCipher(t, "des-cfb")
}

func Test_RC4MD5(t *testing.T) {
	testBlockCipher(t, "rc4-md5")
}

func Test_RC4MD56(t *testing.T) {
	testBlockCipher(t, "rc4-md5-6")
}

func Test_ChaCha20(t *testing.T) {
	testBlockCipher(t, "chacha20")
}

func Test_ChaCha20IETF(t *testing.T) {
	testBlockCipher(t, "chacha20-ietf")
}

func Test_NoneCipher(t *testing.T) {
	testBlockCipher(t, "none")
}
// TODO: 性能测试
