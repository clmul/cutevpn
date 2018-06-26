package cutevpn

import "errors"

var ErrStopLoop = errors.New("stop")
var ErrNoRoute = errors.New("no route to host")

var ErrNoIPv6 = errors.New("IPv6 is not supported")
var ErrInvalidIP = errors.New("invalid IP address")
