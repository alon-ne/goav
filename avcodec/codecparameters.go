package avcodec

//#include <libavcodec/avcodec.h>
import "C"
import (
	"unsafe"
)

func AvcodecParametersToContext(codecContext *Context, codecParameters *CodecParameters) int {
	return int(C.avcodec_parameters_to_context(unsafe.Pointer(codecContext), unsafe.Pointer(codecParameters)))
}

func AvcodecParametersFromContext(codecParameters *CodecParameters, codecContext *Context) int {
	return int(C.avcodec_parameters_from_context(unsafe.Pointer(codecParameters), unsafe.Pointer(codecContext)))
}
