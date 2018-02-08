package cutevpn

import "errors"

var StopLoop = errors.New("stop")
var NoRoute = errors.New("no route to host")

var NoIPv6 = errors.New("IPv6 is not supported")
var InvalidIP = errors.New("invalid IP address")
