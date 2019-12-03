package ss

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

func PrintVersion() {
	const version = "1.1.1"
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

// ss://base64(method:password@host:port)
func CreateURI(method, password, host, port string) (uri string) {
	uri = "ss://"
	raw := method + ":" + password + "@" + host + ":" + port
	str := base64.StdEncoding.EncodeToString([]byte(raw))
	uri = uri + str
	return
}
