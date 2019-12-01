package main

import (
	"encoding/json"
	"flag"
	"github.com/zshorz/shadowsockets/ss"
	"log"
	"os"
	"runtime"
)

var debug bool
var sanitizeIps bool
var udp bool
var managerAddr string
var configFile string
var config *ss.Config

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	var cmdConfig ss.Config
	var printVer, w bool
	var core int

	flag.BoolVar(&printVer, "version", false, "print version")
	flag.StringVar(&configFile, "c", "config.json", "specify config file")
	flag.StringVar(&cmdConfig.Password, "k", "", "password")
	flag.IntVar(&cmdConfig.ServerPort, "p", 0, "server port")
	flag.IntVar(&cmdConfig.Timeout, "t", 300, "timeout in seconds")
	flag.StringVar(&cmdConfig.Method, "m", "", "encryption method, default: aes-256-cfb")
	flag.IntVar(&core, "core", 0, "maximum number of CPU cores to use, default is determinied by Go runtime")
	flag.BoolVar(&debug, "d", false, "print debug message")
	flag.BoolVar(&w, "w", false, "write to config")
	flag.BoolVar(&udp, "u", false, "UDP Relay")
	flag.StringVar(&managerAddr, "manager-address", "", "shadowsocks manager listening address")
	flag.Parse()

	if printVer {
		ss.PrintVersion()
		os.Exit(0)
	}
	ss.SetDebug(debug)

	var err error
	config, err = ss.ParseConfig(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("error reading %s: %v\n", configFile, err)
			os.Exit(1)
		}
		config = &cmdConfig
		ss.UpdateConfig(config, config)
	} else {
		ss.UpdateConfig(config, &cmdConfig)
	}
	if config.Method == "" {
		config.Method = "aes-256-cfb"
	}
	if err = ss.CheckCipherMethod(config.Method); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if core > 0 {
		runtime.GOMAXPROCS(core)
	}
	if w {
		file, err := os.Create(configFile)
		if err != nil {
			log.Printf("can not write to config %s\n", err)
		}
		enc := json.NewEncoder(file)
		enc.SetIndent("", "    ")
		enc.Encode(config)
		file.Close()
	}

	// 同意接口 密码
	unifyPortPassword(config)

	// 启动代理
	if config.PortPassword == nil {
		log.Printf("need Specify port_password or server_port password")
		os.Exit(1)
	}
	for port, password := range config.PortPassword {
		go run(port, password)
		if udp {
			go runUDP(port, password)
		}
	}

	//  TODO: 通过udp连接域服务器对话
	//
	//if managerAddr != "" {
	//	addr, err := net.ResolveUDPAddr("udp", managerAddr)
	//	if err != nil {
	//		log.Println( "Can't resolve address: ", err)
	//		os.Exit(1)
	//	}
	//	conn, err := net.ListenUDP("udp", addr)
	//	if err != nil {
	//		log.Println( "Error listening:", err)
	//		os.Exit(1)
	//	}
	//	log.Printf("manager listening udp addr %v ...\n", managerAddr)
	//	defer conn.Close()
	//	go managerDaemon(conn)
	//}
	waitSignal()
}
