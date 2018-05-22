package socket

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/clmul/cutevpn"
	divert "github.com/clmul/go-windivert"
)

func init() {
	cutevpn.RegisterSocket("divert", openDivert)
}

type divertSocket struct {
	handle divert.Handle
	cancel context.CancelFunc

	addr      divert.Address
	privateIP cutevpn.IPv4
	ifIP      cutevpn.IPv4
}

var wrongFormat = errors.New("wrong format")

func filterIPv4(ip string) (string, error) {
	addr := net.ParseIP(ip)
	if addr == nil {
		return "", wrongFormat
	}
	addr = addr.To4()
	if addr == nil {
		return "", wrongFormat
	}
	return " and ip.DstAddr != " + ip, nil
}

func filterNetwork(cidr string) (string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", wrongFormat
	}
	network := ipnet.IP
	mask := ipnet.Mask
	if len(network) != net.IPv4len || len(mask) != net.IPv4len {
		return "", wrongFormat
	}
	broadcast := make([]byte, net.IPv4len)
	copy(broadcast, network)
	for i := 0; i < net.IPv4len; i++ {
		broadcast[i] |= ^mask[i]
	}
	return fmt.Sprintf(" and (ip.DstAddr < %s or ip.DstAddr > %s)", network, net.IP(broadcast)), nil
}

func filterHost(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	filter := ""
	for _, addr := range addrs {
		filter += " and ip.DstAddr != " + addr
	}
	return filter, nil
}

func makeFilter(ifIdx int, bypass []string) (string, error) {
	filter := fmt.Sprintf("ifIdx = %v and outbound and ip", ifIdx)
	for _, addr := range bypass {
		f, err := filterNetwork(addr)
		if err == nil {
			filter += f
			continue
		}
		f, err = filterIPv4(addr)
		if err == nil {
			filter += f
			continue
		}
		f, err = filterHost(addr)
		if err == nil {
			filter += f
			continue
		}
		return "", err
	}
	log.Println(filter)
	return filter, nil
}

func (d *divertSocket) setIP(localCIDR string) error {
	ip, _, err := net.ParseCIDR(localCIDR)
	if err != nil {
		return err
	}
	ip = ip.To4()
	if ip == nil {
		return errors.New("wrong address")
	}
	copy(d.privateIP[:], ip)
	return nil
}

func openDivert(ctx context.Context, cidr, _, ifce string, bypass []string) (cutevpn.Socket, error) {
	ifIdx, err := strconv.Atoi(ifce)
	if err != nil {
		return nil, err
	}
	filter, err := makeFilter(ifIdx, bypass)
	if err != nil {
		return nil, err
	}
	h, err := divert.Open(filter, divert.LayerNetwork, 0, 0)
	if err != nil {
		return nil, err
	}
	d := &divertSocket{
		cancel: ctx.Value("Cancel").(context.CancelFunc),
		handle: h,
	}
	err = d.setIP(cidr)
	if err != nil {
		return nil, err
	}
	packet := make([]byte, 2048)
	_, d.addr, err = h.Recv(packet)
	if err != nil {
		h.Close()
		return nil, err
	}
	copy(d.ifIP[:], packet[12:16])
	log.Println(d.ifIP)
	cutevpn.RunAfterDone(ctx, h.Close)
	return d, nil
}

func (d *divertSocket) Send(packet []byte) {
	packet = d.nat(packet, divert.DirectionInbound)
	_, err := d.handle.Send(packet, d.addr)
	//log.Println(gopacket.NewPacket(packet, layers.LayerTypeIPv4, gopacket.DecodeOptions{}))
	if err != nil {
		log.Println(err)
		d.cancel()
	}
}

func (d *divertSocket) Recv(packet []byte) int {
	n, _, err := d.handle.Recv(packet)
	if err != nil {
		log.Println(err)
		d.cancel()
		return 0
	}
	packet = d.nat(packet[:n], divert.DirectionOutbound)
	//log.Println(gopacket.NewPacket(packet, layers.LayerTypeIPv4, gopacket.DecodeOptions{}))
	return n
}

func (d *divertSocket) nat(packet []byte, direction uint8) []byte {
	switch direction {
	case divert.DirectionInbound:
		ip := net.IP(packet[16:20])
		if ip.IsGlobalUnicast() {
			copy(packet[16:20], d.ifIP[:])
		}
	case divert.DirectionOutbound:
		ip := net.IP(packet[16:20])
		if ip.IsGlobalUnicast() {
			copy(packet[12:16], d.privateIP[:])
		}
	}
	return divert.CalcChecksums(packet)
}
