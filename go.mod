module github.com/clmul/cutevpn

require (
	github.com/BurntSushi/toml v0.3.0
	github.com/clmul/checksum v0.1.0
	github.com/clmul/go-windivert v0.0.0-20180818034749-0608d894ba22
	github.com/clmul/socks5 v0.0.0-20180327061726-1a1592f2b65e
	github.com/clmul/water v0.0.1
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/go-cmp v0.2.0
	github.com/google/netstack v0.0.0-00010101000000-000000000000
	github.com/miekg/dns v1.1.3 // indirect
	golang.org/x/crypto v0.0.0-20180830192347-182538f80094 // indirect
	golang.org/x/net v0.0.0-20190213061140-3a22650c66bd // indirect
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6 // indirect
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a // indirect
)

replace github.com/google/netstack => github.com/clmul/netstack v0.0.0-20190308035238-c320e3f68db0

go 1.13
