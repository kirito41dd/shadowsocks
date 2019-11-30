package ss

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"time"
)

type Config struct {
	Server       interface{} `json:"server"`
	ServerPort   int         `json:"server_port"`
	LocalPort    int         `json:"local_port"`
	LocalAddress string      `json:"local_address"`
	Password     string      `json:"password"`
	Method       string      `json:"method"`

	// 以下选项只用于 ss server
	PortPassword map[string]string `json:"port_password"`
	Timeout      int               `json:"timeout"`

	// 以下选项只用于 ss client
	// 客户端配置中， 服务器顺序很重要， 因此使用数组而不是map
	ServerPassword [][]string `json:"server_password"` // 多个服务器 [](server, passwd [,method])
}

var readTimeout time.Duration

func (config *Config) GetServerArray() []string {
	// 不建议在服务器选项中指定多个服务器
	// 为了向后兼容，保持现状
	if config.Server == nil {
		return nil
	}

	single, ok := config.Server.(string)
	if ok {
		return []string{single}
	}
	arr, ok := config.Server.([]interface{})
	if ok {
		serverArr := make([]string, len(arr), len(arr))
		for i, s := range arr {
			serverArr[i], ok = s.(string)
			if !ok {
				goto typeError
			}
		}
		return serverArr
	}
typeError:
	panic(fmt.Sprintf("Config.Server type error %v", reflect.TypeOf(config.Server)))
}

func ParseConfig(path string) (config *Config, err error) {
	file, err := os.Open(path) // read access
	if err != nil {
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	config = &Config{}
	if err = json.Unmarshal(data, config); err != nil {
		return
	}
	readTimeout = time.Duration(config.Timeout) * time.Second
	return
}

// 是否启用 Debug , 传 bool
func SetDebug(b bool) {
	isDebug = b
	if b {
		Debug = log.New(os.Stdout, "[DEBUG]", log.Ltime|log.Lshortfile)
	} else {
		Debug = log.New(null, "[DEBUG]", log.Ltime|log.Lshortfile)
	}
}

func UpdateConfig(old, new *Config) {
	// 练习使用 reflect
	newVal := reflect.ValueOf(new).Elem()
	oldVal := reflect.ValueOf(old).Elem()

	for i := 0; i < newVal.NumField(); i++ {
		newField := newVal.Field(i)
		oldField := oldVal.Field(i)

		switch newField.Kind() {
		case reflect.Interface:
			if fmt.Sprintf("%v", newField.Interface()) != "" {
				oldField.Set(newField)
			}
		case reflect.String:
			s := newField.String()
			if s != "" {
				oldField.SetString(s)
			}
		case reflect.Int:
			i := newField.Int()
			if i != 0 {
				oldField.SetInt(i)
			}
		}
	}

	old.Timeout = new.Timeout
	readTimeout = time.Duration(old.Timeout) * time.Second
}
