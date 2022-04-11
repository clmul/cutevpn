package cutevpn

import (
	"encoding/binary"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/vpn"
)

type VPN interface {
	Start() error
	Stop()
	IsRunning() bool
	UpdateGateway(ip string)
	GetNeighbors() Neighbors
}

type Neighbors interface {
	N() int32
	Name(i int32) string
	Addr(i int32) string
}

type neighbors [][2]string

func (ns *neighbors) N() int32 {
	return int32(len(*ns))
}

func (ns *neighbors) Name(i int32) string {
	return (*ns)[i][1]
}

func (ns *neighbors) Addr(i int32) string {
	return (*ns)[i][0]
}

type androidVPN struct {
	*os.File
	*cutevpn.Config
	logFile *os.File
	cvpn    *vpn.VPN
}

func (t *androidVPN) Start() error {
	var err error
	t.cvpn = vpn.NewVPN(t.Config.Name)
	err = vpn.StartWithSocket(t.Config, t.cvpn, t)
	if err != nil {
		t.Close()
		return err
	}
	return nil
}

func (t *androidVPN) Stop() {
	t.logFile.Close()
	t.cvpn.Stop()
	t.Close()
}

func (t *androidVPN) IsRunning() bool {
	select {
	case <-t.cvpn.Done():
		return false
	default:
		return true
	}
}

func (t *androidVPN) UpdateGateway(ip string) {
	t.cvpn.UpdateGateway(ip)
}

func (t *androidVPN) GetNeighbors() Neighbors {
	ns := t.cvpn.Neighbors()
	sort.Slice(ns, func(i, j int) bool {
		addr0 := ns[i].Addr
		addr1 := ns[j].Addr
		return binary.BigEndian.Uint32(addr0[:]) < binary.BigEndian.Uint32(addr1[:])
	})
	r := make(neighbors, len(ns))
	for i, n := range ns {
		r[i] = [2]string{n.Addr.String(), n.Name}
	}
	return &r
}

func Setup(fd int, dir, name, ip, gateway, linksConf string) VPN {
	logFileName := time.Now().UTC().Format("log-20060102-150405.txt")
	logFile, err := os.OpenFile(filepath.Join(dir, logFileName), os.O_WRONLY|os.O_SYNC|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags)
	log.Println("hello")
	var links []string
	for _, link := range strings.Split(linksConf, "\n") {
		link = strings.TrimSpace(link)
		if link != "" {
			links = append(links, link)
		}
	}
	err = unix.SetNonblock(fd, true)
	if err != nil {
		panic(err)
	}
	v := &androidVPN{
		File:    os.NewFile(uintptr(fd), "AndroidVPNService"),
		logFile: logFile,

		Config: &cutevpn.Config{
			Name:         name,
			CIDR:         ip + "/24",
			Gateway:      gateway,
			Links:        links,
			DefaultRoute: false,

			CACert: ``,

			Cert: ``,

			Key: ``,
		},
	}
	return v
}

func (t *androidVPN) Send(packet []byte) {
	_, err := t.Write(packet)
	if err != nil {
		select {
		case <-t.cvpn.Done():
		default:
			panic(err)
		}
	}
}

func (t *androidVPN) Recv(packet []byte) int {
	n, err := t.Read(packet)
	if err != nil {
		select {
		case <-t.cvpn.Done():
		default:
			panic(err)
		}
		return 0
	}
	if packet[0]>>4 != 4 {
		// a non-IPv4 packet
		return 0
	}
	return n
}
