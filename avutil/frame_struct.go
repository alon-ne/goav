package avutil

//#include <libavutil/frame.h>
import "C"

func (f *Frame) SetPts(pts int64) {
	(*C.struct_AVFrame)(f).pts = C.int64_t(pts)
}
