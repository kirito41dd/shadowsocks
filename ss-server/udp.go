package main

import (
	"github.com/zshorz/shadowsocks/ss"
	"log"
	"net"
	"strconv"
)

func runUDP(port, password string) {
	var cipher *ss.Cipher
	port_i, _ := strconv.Atoi(port)
	log.Printf("listening udp port %v\n", port)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv6zero,
		Port: port_i,
	})
	if err != nil {
		log.Printf("error listening udp port %v: %v\n", port, err)
		return
	}
	defer conn.Close()

	cipher, err = ss.NewCipher(config.Method, password)
	if err != nil {
		if debug {
			ss.Debug.Printf("Error generating cipher for udp port: %s %v\n", port, err)
		}
		return
	}
	passwdManger.addUDP(port, password, conn)
	securePacketConn := ss.NewSecurePacketConn(conn, cipher.Copy())
	for {
		// TODO: Traffic 影响效率
		//if err := ss.ReadAndHandleUDPReq(securePacketConn, func(Traffic int) {
		//	passwdManger.addTraffic(port, Traffic)
		//}); err != nil {
		//	if debug {
		//		ss.Debug.Printf("udp read error: %v\n", err)
		//	}
		//	return
		//}
		if err := ss.ReadAndHandleUDPReq(securePacketConn, nil); err != nil {
			if debug {
				ss.Debug.Printf("udp read error: %v\n", err)
			}
			return
		}
	}
}
