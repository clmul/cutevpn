package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"

	"github.com/BurntSushi/toml"
	"golang.org/x/sys/unix"

	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/vpn"
	"github.com/clmul/socks5"
)

var conf string

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.StringVar(&conf, "config", "config.toml", "config file path")
	flag.Parse()
	//disableDefaultDNS()
}

func disableDefaultDNS() {
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, errors.New("disable default DNS")
	}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial:     dial,
	}
}

type Config struct {
	cutevpn.Config
	SOCKS5Server string
	HTTPServer   string
	Started      string
	Stopped      string
}

func defaultConf(conf *Config) {
	if conf.Socket == "" {
		conf.Socket = "tun"
	}
	if conf.MTU == 0 {
		conf.MTU = 1350
	}
}

func main() {
	conf, err := parseConfigFile(conf)
	if err != nil {
		log.Fatal(err)
	}
	defaultConf(conf)
	v, err := vpn.Start(&conf.Config)
	if err != nil {
		log.Fatal(err)
	}
	if conf.HTTPServer != "" {
		v.StartHTTP(conf.HTTPServer)
	}

	if conf.Started != "" {
		bash(conf.Started)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, unix.SIGTERM)

	if conf.SOCKS5Server != "" {
		go socks5Server(conf.SOCKS5Server)
	}

	<-c
	log.Println("received SIGINT")

	if conf.HTTPServer != "" {
		v.StopHTTP()
	}
	v.Stop()
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
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var conf Config
	err = toml.Unmarshal(d, &conf)
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
