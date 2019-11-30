package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/zshorz/shadowsockets/ss"
	"log"
	"os"
	"path"
	"strconv"
)

var debug bool

// ss-local 目前只支持 tcp 代理
func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	var configFile, cmdServer, cmdURI string
	var cmdConfig ss.Config
	var printVer, w bool

	flag.BoolVar(&printVer, "version", false, "print version")
	flag.StringVar(&configFile, "c", "config.json", "specify config file")
	flag.StringVar(&cmdServer, "s", "", "server address")
	flag.StringVar(&cmdConfig.LocalAddress, "b", "", "local address, listen only to this address if specified")
	flag.StringVar(&cmdConfig.Password, "k", "", "password")
	flag.IntVar(&cmdConfig.ServerPort, "p", 0, "server port")
	flag.IntVar(&cmdConfig.Timeout, "t", 300, "timeout in seconds")
	flag.IntVar(&cmdConfig.LocalPort, "l", 0, "local socks5 proxy port")
	flag.StringVar(&cmdConfig.Method, "m", "", "encryption method, default: aes-256-cfb")
	flag.BoolVar(&debug, "d", false, "print debug message")
	flag.BoolVar(&w, "w", false, "write to config")
	flag.StringVar(&cmdURI, "u", "", "shadowsocks URI")
	flag.Parse()

	if printVer {
		ss.PrintVersion()
		os.Exit(0)
	}
	ss.SetDebug(debug)
	if s, e := parseURI(cmdURI, &cmdConfig); e != nil {
		log.Printf("invalid URI: %s\n", e.Error())
		flag.Usage()
		os.Exit(1)
	} else if s != "" {
		cmdServer = s
	}
	cmdConfig.Server = cmdServer

	// 没有配置文件 尝试在执行bin目录寻找
	exists, err := ss.IsFileExist(configFile)
	binDir := path.Dir(os.Args[0])
	if (!exists || err != nil) && binDir != "" && binDir != "." {
		oldConfig := configFile
		configFile = path.Join(binDir, "config.json")
		log.Printf("%s not found, try config file %s\n", oldConfig, configFile)
	}

	config, err := ss.ParseConfig(configFile)
	if err != nil {
		config = &cmdConfig
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", configFile, err)
			os.Exit(1)
		}
	} else {
		ss.UpdateConfig(config, &cmdConfig)
	}
	if config.Method == "" {
		config.Method = "aes-256-cfb"
	}
	if w {
		file, err := os.Create(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "can not write to config %s\n", err)
		}
		enc := json.NewEncoder(file)
		enc.SetIndent("", "    ")
		enc.Encode(config)
		file.Close()
	}

	if len(config.ServerPassword) == 0 { // 单服务器
		if !enoughOptions(config) {
			fmt.Fprintln(os.Stderr, "must specify server address, password and both server/local port")
			os.Exit(1)
		}
	} else { // 可能多服务器
		if config.Password != "" || config.ServerPort != 0 || config.GetServerArray() != nil {
			fmt.Fprintln(os.Stderr, "given server_password, ignore server, server_port and password option:", config)
		}
		if config.LocalPort == 0 {
			fmt.Fprintln(os.Stderr, "must specify local port")
			os.Exit(1)
		}
	}

	parseServerConfig(config)

	run(config.LocalAddress + ":" + strconv.Itoa(config.LocalPort))
}
