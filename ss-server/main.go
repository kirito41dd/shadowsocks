package main

import (
	"encoding/json"
	"flag"
	"github.com/zshorz/ezlog"
	"github.com/zshorz/shadowsocks/ss"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"runtime"
)

var debug bool
var sanitizeIps bool
var udp bool
var managerAddr string
var configFile string
var config *ss.Config
var printURI bool // 自动获取 公网ip
var domain string // 如果指定，将阻止获取公网ip,用指定的domain/ip代替生成uri

func main() {
	ezlog.SetOutput(os.Stdout)
	ezlog.SetFlags(ezlog.BitDefault)

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
	flag.BoolVar(&printURI, "uri", false, "print URI, auto get public ip")
	flag.StringVar(&domain, "domain", "", "domain/ip instead public ip, when generate uri")
	flag.BoolVar(&sanitizeIps, "sanitize", false, "on debug, sanitize ip:port to x.x.x.x:zzzz")
	flag.Parse()

	if printVer {
		ss.PrintVersion()
		os.Exit(0)
	}
	ss.SetDebug(debug)

	var err error
	// 没有配置文件 尝试在执行bin目录寻找
	exists, err := ss.IsFileExist(configFile)
	binDir := path.Dir(os.Args[0])
	if (!exists || err != nil) && binDir != "" && binDir != "." {
		oldConfig := configFile
		configFile = path.Join(binDir, "config.json")
		ezlog.Infof("%s not found, try config file %s\n", oldConfig, configFile)
	}
	config, err = ss.ParseConfig(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			ezlog.Errorf("error reading %s: %v\n", configFile, err)
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
		ezlog.Error(err)
		os.Exit(1)
	}
	if core > 0 {
		runtime.GOMAXPROCS(core)
	}
	if w {
		file, err := os.Create(configFile)
		if err != nil {
			ezlog.Errorf("can not write to config %s\n", err)
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
		ezlog.Info("need Specify port_password or server_port password")
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

	// 性能分析
	if debug {
		go func() {
			addr := "127.0.0.1:9999"
			// http://127.0.0.1:9999/debug/pprof/
			if err := http.ListenAndServe(addr, nil); err != nil {
				ezlog.Error("start pprof failed on %s\n", addr)
			} else {
				ezlog.Info("you can look pprof at http://127.0.0.1:9999/debug/pprof/\n")
			}
		}()
	}

	waitSignal()
}
