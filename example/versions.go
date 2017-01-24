package main

import (
	"github.com/alon-ne/goav/avcodec"
	"github.com/alon-ne/goav/avdevice"
	"github.com/alon-ne/goav/avfilter"
	"github.com/alon-ne/goav/avformat"
	"github.com/alon-ne/goav/avutil"
	"github.com/alon-ne/goav/swresample"
	"github.com/alon-ne/goav/swscale"
	"log"
)

func main() {

	// Register all formats and codecs
	avformat.AvRegisterAll()
	avcodec.AvcodecRegisterAll()

	log.Printf("AvFilter Version:\t%v", avfilter.AvfilterVersion())
	log.Printf("AvDevice Version:\t%v", avdevice.AvdeviceVersion())
	log.Printf("SWScale Version:\t%v", swscale.SwscaleVersion())
	log.Printf("AvUtil Version:\t%v", avutil.AvutilVersion())
	log.Printf("AvCodec Version:\t%v", avcodec.AvcodecVersion())
	log.Printf("Resample Version:\t%v", swresample.SwresampleLicense())

}
