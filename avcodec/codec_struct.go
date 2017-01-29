package avcodec

//#include <libavcodec/avcodec.h>
import "C"

func (c *Codec) Capabilities() int {
	return int(c.capabilities)
}

func (c *Codec) FrameSize() int {
	return int(c.frame_size)
}