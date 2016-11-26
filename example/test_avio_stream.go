package main

import (
	"fmt"
	"github.com/giorgisio/goav/avformat"
)

type TestStream struct {
}

func (s *TestStream) ReadPacket(buf avformat.AvIOPacket, bufSize int) int {
	buf[0] = 5
	buf[1] = 4
	buf[2] = 3
	return 3
}

func (s *TestStream) WritePacket(buf avformat.AvIOPacket, bufSize int) int {
	return 0
}

func (s *TestStream) Seek(offset int64, whence int) int64 {
	return 0
}

func main() {
	bufferSize := 4096
	stream := TestStream{}
	avformat.AvIOAllocContext(bufferSize, 0, &stream)
	fmt.Printf("Hi there\n")
}
