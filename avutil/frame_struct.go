package avutil

//#include <libavutil/frame.h>
import "C"

func (f *Frame) Pts() int64 {
	return int64(f.pts)
}
func (f *Frame) SetPts(pts int64) {
	(*C.struct_AVFrame)(f).pts = C.int64_t(pts)
}

func (f *Frame) SetFormat(format int) {
	f.format = C.int(format)
}

func (f *Frame) SetWidth(width int) {
	f.width = C.int(width)
}

func (f *Frame) SetHeight(height int) {
	f.height = C.int(height)
}
