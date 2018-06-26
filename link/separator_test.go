package link

import (
	"bufio"
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"
)

func TestSeparator(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	r, w := io.Pipe()
	c := make(chan []byte, 16)
	go func() {
		for i := 0; i < 16384; i++ {
			size := rand.Intn(32768)
			buffer := make([]byte, size)
			n, err := rand.Read(buffer)
			if n != size || err != nil {
				panic(err)
			}
			c <- buffer
			w.Write(addSep(buffer))
		}
		close(c)
	}()
	scanner := bufio.NewScanner(r)
	scanner.Split(split)
	for buf := range c {
		if !scanner.Scan() {
			panic(scanner.Err())
		}
		got := scanner.Bytes()
		if !bytes.Equal(got, buf) {
			t.Errorf("got %v, want %v", got, buf)
		}
	}
}
