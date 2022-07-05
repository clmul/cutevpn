package encryption

type Plain struct{}

func (a Plain) Encrypt(packet []byte) []byte {
	return packet
}

func (a Plain) Decrypt(packet []byte) ([]byte, error) {
	return packet, nil
}

func (a Plain) Overhead() int {
	return 0
}
