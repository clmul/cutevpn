package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"

	"github.com/clmul/cutevpn"
)

type AESGCM struct {
	cipher cipher.AEAD
}

func NewAESGCM(secret string) (cutevpn.Cipher, error) {
	key, err := hex.DecodeString(secret)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	a := AESGCM{cipher: aead}
	return a, nil
}

func (a AESGCM) Encrypt(packet []byte) []byte {
	nonce := make([]byte, a.cipher.NonceSize())
	_, err := rand.Read(nonce)
	if err != nil {
		log.Fatal(err)
	}
	packet = a.cipher.Seal(packet[:0], nonce, packet, nil)
	return append(packet, nonce...)
}

func (a AESGCM) Decrypt(packet []byte) ([]byte, error) {
	var err error
	ns := a.cipher.NonceSize()
	if len(packet) < ns {
		return nil, errors.New("packet is too short")
	}
	packet, nonce := packet[:len(packet)-ns], packet[len(packet)-ns:]
	packet, err = a.cipher.Open(packet[:0], nonce, packet, nil)
	return packet, err
}

func (a AESGCM) Overhead() int {
	return a.cipher.Overhead() + a.cipher.NonceSize()
}
