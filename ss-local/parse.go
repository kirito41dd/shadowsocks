package main

import (
	"bytes"
	"encoding/base64"
	"github.com/zshorz/shadowsockets/ss"
	"strconv"
	"strings"
)

// 返回 host  其余自动设置
func parseURI(u string, cfg *ss.Config) (string, error) {
	if u == "" {
		return "", nil
	}
	// ss://base64(method:password)@host:port
	// ss://base64(method:password@host:port)
	u = strings.TrimLeft(u, "ss://")
	i := strings.IndexRune(u, '@')
	var headParts, tailParts [][]byte
	if i == -1 { // 第二种形式
		dat, err := base64.StdEncoding.DecodeString(u)
		if err != nil {
			return "", err
		}
		parts := bytes.Split(dat, []byte("@"))
		if len(parts) != 2 {
			return "", invalidURI
		}
		headParts = bytes.SplitN(parts[0], []byte(":"), 2)
		tailParts = bytes.SplitN(parts[1], []byte(":"), 2)

	} else { // 第一种形式
		if 1+1 >= len(u) {
			return "", invalidURI
		}
		tailParts = bytes.SplitN([]byte(u[i+1:]), []byte(":"), 2)
		dat, err := base64.StdEncoding.DecodeString(u[:i])
		if err != nil {
			return "", err
		}
		headParts = bytes.SplitN(dat, []byte(":"), 2)
	}

	if len(headParts) != 2 || len(tailParts) != 2 {
		return "", invalidURI
	}

	cfg.Method = string(headParts[0])
	cfg.Password = string(headParts[1])

	p, e := strconv.Atoi(string(tailParts[1]))
	if e != nil {
		return "", e
	}
	cfg.ServerPort = p
	return string(tailParts[0]), nil
}
