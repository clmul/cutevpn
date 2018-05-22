package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"

	"github.com/BurntSushi/toml"
	"github.com/clmul/socks5"

	"github.com/clmul/cutevpn"
	_ "github.com/clmul/cutevpn/cipher"
	_ "github.com/clmul/cutevpn/link"
	_ "github.com/clmul/cutevpn/ospf"
	_ "github.com/clmul/cutevpn/socket"
)

var conf string

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.StringVar(&conf, "config", "config.toml", "config file path")
	flag.Parse()
}

type Config struct {
	cutevpn.Config
	SOCKS5Server string
	HTTPServer   string
	Started      string
	Stopped      string
	Comment      string
}

func defaultConf(conf *Config) {
	if conf.Socket == "" {
		conf.Socket = "tun"
	}
	if conf.MTU == 0 {
		conf.MTU = 1400
	}
	if conf.Routing == "" {
		conf.Routing = "ospf"
	}
	for i := range conf.Links {
		if conf.Links[i].Link == "" {
			conf.Links[i].Link = "udp4"
		}
	}
}

func main() {
	conf, err := parseConfigFile(conf)
	if err != nil {
		log.Fatal(err)
	}
	defaultConf(conf)
	vpn, err := conf.Start()
	if err != nil {
		log.Fatal(err)
	}
	if conf.HTTPServer != "" {
		vpn.StartHTTP(conf.HTTPServer)
	}

	if conf.Started != "" {
		bash(conf.Started)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	if conf.SOCKS5Server != "" {
		go socks5Server(conf.SOCKS5Server)
	}

	<-c
	log.Println("received SIGINT")

	if conf.HTTPServer != "" {
		vpn.StopHTTP()
	}
	vpn.Stop()
	if conf.Stopped != "" {
		bash(conf.Stopped)
	}
}

func bash(script string) {
	cmd := exec.Command("bash", "-x")
	cmd.Stdin = bytes.NewBufferString(script)
	output, err := cmd.CombinedOutput()
	log.Println("\n" + string(output))
	if err != nil {
		log.Println(err)
	}
}

func parseConfigFile(filename string) (*Config, error) {
	var conf Config
	meta, err := toml.DecodeFile(filename, &conf)
	if err != nil {
		return nil, err
	}
	undecoded := meta.Undecoded()
	if len(undecoded) > 0 {
		return nil, fmt.Errorf("unrecognized keys in config: %v", undecoded)
	}
	return &conf, err
}

func socks5Server(addr string) {
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(server.ListenAndServe("tcp", addr))
}
