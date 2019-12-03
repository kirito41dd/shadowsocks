package main

import (
	"errors"
	"github.com/zshorz/shadowsocks/ss"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func waitSignal() {
	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)
	for sig := range sigChan {
		if sig == syscall.SIGHUP {
			updatePasswd()
		} else {
			log.Printf("caught signal %v, exit", sig)
			os.Exit(0)
		}
	}
}

// 是否保护ip
func sanitizeAddr(addr net.Addr) string {
	if sanitizeIps {
		return "x.x.x.x:zzzz"
	} else {
		return addr.String()
	}
}

func unifyPortPassword(config *ss.Config) (err error) {
	if len(config.PortPassword) == 0 {
		if !enoughOptions(config) {
			log.Println("must specify both port and password")
			return errors.New("not enough options")
		}
		port := strconv.Itoa(config.ServerPort)
		config.PortPassword = map[string]string{port: config.Password}
	} else {
		if config.Password != "" || config.ServerPort != 0 {
			log.Println("given port_password, ignore server_port and password option")
		}
	}
	return
}

func enoughOptions(config *ss.Config) bool {
	return config.ServerPort != 0 && config.Password != ""
}
