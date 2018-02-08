package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	"github.com/BurntSushi/toml"
	"github.com/armon/go-socks5"

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
	Socks5Server bool
}

func main() {
	conf, err := parseConfigFile(conf)
	if err != nil {
		log.Fatal(err)
	}
	vpn, err := conf.Start()
	if err != nil {
		log.Fatal(err)
	}
	vpn.StartHTTP()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	if conf.Socks5Server {
		go socks5Server(fmt.Sprintf("%s:%d", vpn.IP(), 1080))
	}

	<-c
	log.Println("received SIGINT")

	vpn.StopHTTP()
	vpn.Stop()
}

func parseConfigFile(filename string) (*Config, error) {
	var conf Config
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = toml.Unmarshal(content, &conf)
	if err != nil {
		return nil, err
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
