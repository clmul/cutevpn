# CuteVPN

## What is CuteVPN

There are always some difficulties making connections go through the border of PRC.

1. UDP hardly works in PRC.
2. Some IPs(those from blocked websites, like Google) are not reachable in PRC.
3. There are fake DNS replies.
4. TCP connections often get interrupted, which makes [Shadowsocks](https://github.com/shadowsocks) not reliable.

CuteVPN was written to overcome these problems. It makes computers across the border connectable.
I wrote CuteVPN just because the firewall, but the VPN is a decentralized Mesh VPN now.
For personal use case, It works well as an alternate to Tailscale or ZeroTier.

## How does it work

A minimal IP protocol and routing protocol are implemented by CuteVPN.
Failover and load balancing are supported by the routing protocol.

A subnet is established by point-to-point links. The links are configured manually.
Any two nodes in the subnet will be reachable.
Any node can work as a network exit for any nodes in the subnet.

Some ICMP protocols are also implemented so `ping` and `traceroute` works on the subnet.

And any nodes in the subnet can act like a gateway for other nodes.

A typical use case is like this.
```
            A ----- B ----- C                       the PRC
           / \             /
===============================================     the border
         /     \         /
        H ----- J ----- K                           outside the PRC
         \     /
          \   /
            Z
   
```

If B wants to connect to Z, there are many possible routes.
```
B - A - H - Z
B - A - J - Z
B - C - K - J - Z
B - C - K - J - A - H - Z
```

The routing protocol is a simplified OSPF protocol.
CuteVPN will sort the routes with their round-trip times and choose the shortest one.

If a link is interrupted by the firewall, it will detect it and choose another one. (failover)

If there is another route whose round-trip time is nearly the same as the shortest one, this route will also be used to send packets. (load balancing)

## Build

```shell
git clone https://github.com/clmul/cutevpn
cd cutevpn/cutevpn
go build
```

## Usage

```shell
sudo ./cutevpn -config config.toml
```

## Config

The format is [TOML](https://github.com/toml-lang/toml)

An example config file with document is `cutevpn/config.toml`
