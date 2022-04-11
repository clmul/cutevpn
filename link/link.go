package link

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"

	"github.com/clmul/cutevpn"
)

func New(vpn cutevpn.VPN, linkURL *url.URL, cacert, cert, key string) (cutevpn.Link, error) {
	switch linkURL.Scheme {
	case "tls":
		certificate, err := tls.X509KeyPair([]byte(cert), []byte(key))
		if err != nil {
			return nil, err
		}
		ca := x509.NewCertPool()
		if !ca.AppendCertsFromPEM([]byte(cacert)) {
			return nil, fmt.Errorf("can't read CA certificate")
		}
		return newTLS(vpn, linkURL, certificate, ca)
	default:
		return nil, fmt.Errorf("unknown link %s", linkURL.Scheme)
	}
}
