package ss

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func PrintVersion() {
	const version = "1.2.4"
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

var publicIp = ""

func GetPublicIP() string {
	if publicIp != "" {
		return publicIp
	}
	resp, err := http.Get("http://ipinfo.io/ip")
	defer resp.Body.Close()
	if err != nil {
		return ""
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	publicIp = string(data)
	publicIp = strings.Replace(publicIp, "\n", "", -1) // 去除行末空格，否则生成uri会出错
	return publicIp
}
