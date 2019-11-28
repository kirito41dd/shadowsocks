package ss

import (
	"crypto/hmac"
	"crypto/sha1"
	"errors"
	"fmt"
	"os"
)

func PrintVersion() {
	const version = "0.0.0"
	fmt.Println("shadowsocks version", version)
}

// 检查常规文件是否存在， 常规文件不是
// ModeDir | ModeSymlink | ModeNamedPipe | ModeSocket | ModeDevice | ModeCharDevice | ModeIrregular
func IsFileExist(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		if stat.Mode()&os.ModeType == 0 {
			return true, nil
		}
		return false, errors.New(path + " exists but not regular file")
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func HmacSha1(key []byte, data []byte) []byte {
	hmacSha1 := hmac.New(sha1.New, key)
	hmacSha1.Write(data)
	return hmacSha1.Sum(nil)[:10]
}
