package avformat

/*

//TODO: Remove unsued includes from below

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
extern long long seekStream(void* opaque, long long offset, int whence);

static inline AVIOContext* avio_alloc_context_wrapper(unsigned char* buffer, int bufferSize, int writeFlag, int streamIndex)
{
	void* opaque = (void*)(long long)streamIndex;

	//TEMP TEMP TEMP
	FILE* f = fopen("/Users/alonne/temp/c.txt", "w");
	int r = readStreamPacket(opaque, buffer, 3);
	fprintf(f, "avio_alloc_context_wrapper: buf=");
	for (int i = 0; i < 3; ++i)
	{
		fprintf(f, "%d ", buffer[i]);
	}
	fprintf(f, "\n");
	fprintf(f, "avio_alloc_context_wrapper: returned %d\n", r);
	fclose(f);
	//TEMP TEMP TEMP

	return avio_alloc_context(buffer, bufferSize, writeFlag, opaque, &readStreamPacket, &writeStreamPacket, seekStream);
}
*/
import "C"
import (
	"unsafe"
	"sync"
	"github.com/giorgisio/goav/avutil"
)

const maxArraySize = 1 << 31 - 1

type AvIOPacket *[maxArraySize]uint8

type AvIOStream interface {
	ReadPacket(buf AvIOPacket, bufSize int) int
	WritePacket(buf AvIOPacket, bufSize int) int
	Seek(offset int64, whence int) int64
}


var avioStreams map[int]AvIOStream = make(map[int]AvIOStream)
var lastStreamIndex int
var avioStreamsMutex sync.Mutex

func registerAvIOStream(stream AvIOStream) int {
	avioStreamsMutex.Lock()
	defer avioStreamsMutex.Unlock()
	streamIndex := lastStreamIndex
	lastStreamIndex++
	avioStreams[streamIndex] = stream
	return streamIndex
}

func unregisterAvIOStream(streamIndex int) {
	avioStreamsMutex.Lock()
	defer avioStreamsMutex.Unlock()
	delete(avioStreams, streamIndex)
}

func getAvIOStreamByIndex(index int) AvIOStream {
	avioStreamsMutex.Lock()
	defer avioStreamsMutex.Unlock()
	return avioStreams[index]
}

func getAvIOStreamByOpaque(opaque unsafe.Pointer) AvIOStream {
	index := int(uintptr(opaque))
	return getAvIOStreamByIndex(index)
}

//export readStreamPacket
func readStreamPacket(opaque unsafe.Pointer, buf *C.uchar, bufSize C.int) C.int {
	stream := getAvIOStreamByOpaque(opaque)
	goBuf := AvIOPacket(unsafe.Pointer(buf))
	return C.int(stream.ReadPacket(goBuf, int(bufSize)))
}

//export writeStreamPacket
func writeStreamPacket(opaque unsafe.Pointer, buf *C.uchar, bufSize C.int) C.int {
	stream := getAvIOStreamByOpaque(opaque)
	goBuf := AvIOPacket(unsafe.Pointer(buf))
	return C.int(stream.WritePacket(goBuf, int(bufSize)))
}

//export seekStream
func seekStream(opaque unsafe.Pointer, offset C.longlong, whence C.int) C.longlong {
	stream := getAvIOStreamByOpaque(opaque)
	return C.longlong(stream.Seek(int64(offset), int(whence)))
}

func AvIOAllocContext(bufferSize int, writeFlag int, stream AvIOStream) (*AvIOContext, int) {
	buffer := (*C.uchar)(avutil.AvMalloc(uintptr(bufferSize)))
	streamIndex := registerAvIOStream(stream)
	context := (*AvIOContext)(C.avio_alloc_context_wrapper(buffer, C.int(bufferSize), C.int(writeFlag), C.int(streamIndex)))
	return context, streamIndex
}

func AvIODeallocateContext(context *AvIOContext, streamIndex int) {
	unregisterAvIOStream(streamIndex)
	avutil.AvFree(unsafe.Pointer(context.buffer))
	avutil.AvFree(unsafe.Pointer(context))
}