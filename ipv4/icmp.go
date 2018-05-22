package ipv4

import (
	"encoding/binary"
	"github.com/clmul/checksum"
)

const (
	ICMP = 1
	TCP  = 6
	UDP  = 17

	IPHeaderLen   = 20
	ICMPHeaderLen = 8

	ICMPTimeExceeded = 11

	IPv4VersionIHL        = 0x45
	IPv4TotalLengthOffset = 2
	IPv4TimeToLiveOffset  = 8
	IPv4ProtocolOffset    = 9
	IPv4SourceOffset      = 12
	IPv4DestinationOffset = 16
)

func TimeExceeded(from, to [4]byte, packet []byte) []byte {
	ipv4HeaderLen := int(packet[0]&0xf) * 4
	icmpLen := IPHeaderLen + ICMPHeaderLen
	if len(packet) >= ipv4HeaderLen+8 {
		icmpLen += ipv4HeaderLen + 8
	} else {
		icmpLen += len(packet)
	}

	result := make([]byte, icmpLen)
	copy(result[IPHeaderLen+ICMPHeaderLen:], packet)
	result[IPHeaderLen] = ICMPTimeExceeded
	fillIPHeader(ICMP, from, to, result)
	checksum.Calc(result)
	return result
}

func fillIPHeader(protocol uint8, from, to [4]byte, packet []byte) {
	packet[0] = IPv4VersionIHL
	binary.BigEndian.PutUint16(packet[IPv4TotalLengthOffset:], uint16(len(packet)))
	packet[IPv4TimeToLiveOffset] = 64
	packet[IPv4ProtocolOffset] = protocol
	copy(packet[IPv4SourceOffset:], from[:])
	copy(packet[IPv4DestinationOffset:], to[:])
}
