package link

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"

	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/encryption"
)

func New(vpn cutevpn.VPN, linkURL *url.URL) error {
	var cipher cutevpn.Cipher = encryption.Plain{}
	if secret := linkURL.Query().Get("secret"); secret != "" {
		var err error
		cipher, err = encryption.NewAESGCM(secret)
		if err != nil {
			return err
		}
	}
	switch linkURL.Scheme {
	case "tls":
		certFile := linkURL.Query().Get("cert")
		keyFile := linkURL.Query().Get("key")
		caCertFile := linkURL.Query().Get("cacert")
		certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
		ca := x509.NewCertPool()
		cacert, err := os.ReadFile(caCertFile)
		if err != nil {
			return err
		}
		if !ca.AppendCertsFromPEM(cacert) {
			return fmt.Errorf("can't read CA certificate")
		}
		return newTLS(vpn, linkURL, certificate, ca)
	case "ipip":
		return newIPIP(vpn, linkURL, cipher)
	case "udp":
		return newUDP(vpn, linkURL, cipher)
	default:
		return fmt.Errorf("unknown link %s", linkURL.Scheme)
	}
}
