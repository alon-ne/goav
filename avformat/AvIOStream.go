package avformat

/*

//TODO: Remove unused includes from below

#cgo pkg-config: libavformat libavcodec libavutil libavdevice libavfilter libswresample libswscale
#include <stdio.h>
#include <stdlib.h>
#include <inttypes.h>
#include <stdint.h>
#include <string.h>
#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavutil/opt.h>
#include <libavdevice/avdevice.h>

extern int readStreamPacket(void* opaque, unsigned char* buf, int bufSize);
extern int writeStreamPacket(void* opaque, unsigned char* buf, int bufSize);
extern int64_t seekStream(void* opaque, int64_t offset, int whence);

static inline int readStreamPacketWrapper(void* opaque, unsigned char* buf, int bufSize)
{
	FILE* outFile = fopen("/home/alon/outFileCgo", "a");
	int readBytes;
	readBytes = readStreamPacket(opaque, buf, bufSize);
	fwrite(buf, 1, readBytes, outFile);
	fclose(outFile);
	return readBytes;
}

static inline AVIOContext* avio_alloc_context_wrapper(unsigned char* buffer, int bufferSize, int writeFlag, int streamIndex)
{
	void* opaque = (void*)(long long)streamIndex;

	return avio_alloc_context(
		buffer,
		bufferSize,
		writeFlag,
		opaque,
		&readStreamPacketWrapper,
		NULL, //&writeStreamPacket,
		NULL); //&seekStream);
}
*/
import "C"
import (
	"unsafe"
	"sync"
	"github.com/alon-ne/goav/avutil"
	"fmt"
	"errors"
)

const (
	maxArraySize = 1 << 31 - 1
	AVIO_FLAG_WRITE = 2
)


type AvIOPacket *[maxArraySize]uint8

func init() {
	AvRegisterAll()
}

type AvIOStream interface {
	ReadPacket(buf AvIOPacket, bufSize int) int
	WritePacket(buf AvIOPacket, bufSize int) int
	Seek(offset int64, whence int) int64
}


var avioStreams map[int]AvIOStream = make(map[int]AvIOStream)
var lastStreamIndex int

type dummyMutex struct {

}

func (*dummyMutex) Lock() {

}

func (*dummyMutex) Unlock() {

}

var avioStreamsMutex sync.Mutex
//var avioStreamsMutex dummyMutex

func registerAvIOStream(stream AvIOStream) int {
	fmt.Printf("Entering avioStreamsMutex\n")
	avioStreamsMutex.Lock()
	defer avioStreamsMutex.Unlock()
	streamIndex := lastStreamIndex
	lastStreamIndex++
	avioStreams[streamIndex] = stream
	fmt.Printf("Leaving avioStreamsMutex\n")
	return streamIndex
}

func unregisterAvIOStream(streamIndex int) {
	fmt.Printf("Entering avioStreamsMutex\n")
	avioStreamsMutex.Lock()
	defer avioStreamsMutex.Unlock()
	delete(avioStreams, streamIndex)
	fmt.Printf("Leaving avioStreamsMutex\n")
}

func getAvIOStreamByIndex(index int) AvIOStream {
	fmt.Printf("Entering avioStreamsMutex\n")
	avioStreamsMutex.Lock()
	defer avioStreamsMutex.Unlock()
	stream, ok := avioStreams[index]
	if (!ok) {
		fmt.Printf("Failed to find stream with index %d\n", index)
		return nil
	}
	fmt.Printf("Leaving avioStreamsMutex\n")
	return stream
}

func getAvIOStreamByOpaque(opaque unsafe.Pointer) AvIOStream {
	index := int(uintptr(opaque))
	return getAvIOStreamByIndex(index)
}

//export readStreamPacket
func readStreamPacket(opaque unsafe.Pointer, buf *C.uchar, bufSize C.int) C.int {

	stream := getAvIOStreamByOpaque(opaque)
	if stream == nil {
		return -1
	}
	goBuf := AvIOPacket(unsafe.Pointer(buf))
	bytesRead := C.int(stream.ReadPacket(goBuf, int(bufSize)))
	fmt.Printf("readStreamPacket: bufSize=%d, returning %d\n", int(bufSize), int(bytesRead))

	return bytesRead
}

//export writeStreamPacket
func writeStreamPacket(opaque unsafe.Pointer, buf *C.uchar, bufSize C.int) C.int {
	stream := getAvIOStreamByOpaque(opaque)
	if stream == nil {
		return -1
	}
	goBuf := AvIOPacket(unsafe.Pointer(buf))
	return C.int(stream.WritePacket(goBuf, int(bufSize)))
}

//export seekStream
func seekStream(opaque unsafe.Pointer, offset C.int64_t, whence C.int) C.int64_t {
	stream := getAvIOStreamByOpaque(opaque)
	if stream == nil {
		return -1
	}
	return C.int64_t(stream.Seek(int64(offset), int(whence)))
}

func AvIOAllocContext(bufferSize int, writeFlag int, stream AvIOStream) (*AvIOContext, int, error) {
	buffer := avutil.AvMalloc(uintptr(bufferSize))
	if buffer == nil {
		return nil,-1,errors.New("Failed to allocate buffer")
	}
	streamIndex := registerAvIOStream(stream)
	context := (*AvIOContext)(C.avio_alloc_context_wrapper((*C.uchar)(buffer), C.int(bufferSize), C.int(writeFlag), C.int(streamIndex)))
	if context == nil {
		unregisterAvIOStream(streamIndex)
		return nil,-1,errors.New("Failed to allocate avio context")
	}
	return context, streamIndex, nil
}

func AvIODeallocateContext(context *AvIOContext, streamIndex int) {
	unregisterAvIOStream(streamIndex)
	avutil.AvFree(unsafe.Pointer(context.buffer))
	avutil.AvFree(unsafe.Pointer(context))
}

func AvIOOpen(context **AvIOContext, filename string, flags int) int {
	return int(C.avio_open((**C.struct_AVIOContext)(unsafe.Pointer(context)), C.CString(filename), C.int(flags)))
}