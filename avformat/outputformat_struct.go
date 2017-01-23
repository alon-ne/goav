package avformat

//#cgo pkg-config: libavformat
//#include <libavformat/avformat.h>
import "C"

func (o *OutputFormat) VideoCodec() int {
	return int(o.video_codec)
}

func (o *OutputFormat) SetVideoCodec(videoCodec int) {
	o.video_codec = C.enum_AVCodecID(videoCodec)
}

func (o *OutputFormat) Flags() int {
	return int(o.flags)
}

func (o *OutputFormat) SetFlags(flags int) {
	o.flags = C.int(flags)
}