package main

import (
	"errors"
	"github.com/zshorz/shadowsocks/ss"
	"math/rand"
	"time"
)

var (
	invalidURI       = errors.New("invalid URI")
	errAddrType      = errors.New("socks addr type not supported")
	errVer           = errors.New("socks version not supported")
	errMethod        = errors.New("socks only support 1 method now")
	errAuthExtraData = errors.New("socks authentication get extra data")
	errReqExtraData  = errors.New("socks request get extra data")
	errCmd           = errors.New("socks command not supported")
)

func init() {
	rand.Seed(time.Now().Unix())
}

func enoughOptions(config *ss.Config) bool {
	return config.Server != nil && config.ServerPort != 0 &&
		config.LocalPort != 0 && config.Password != ""
}

func Traffic(n int) {
	ss.Debug.Printf("%d bytes data exchange\n", n)
}
