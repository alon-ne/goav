package main

import (
	"fmt"
	"errors"
	"github.com/alon-ne/goav/avutil"
	"github.com/alon-ne/goav/avformat"
	"github.com/alon-ne/goav/avlog"
	"github.com/alon-ne/goav/avcodec"
)

const (
	outputFormatName = "mp4"
	outputCodecThreadCount = 4
	outputCodecBitRate = 300000
	outputStreamFrameRate = 30
	outputCodecGopSize = 12
	outputStreamPixFmt = avcodec.AV_PIX_FMT_YUV420P
	outputFrameAlignment = 32
)

func checkAvRet(funcName string, avfunc func() int) bool {
	if errnum := avfunc(); errnum < 0 {
		errorMessage := avutil.AvStrError(errnum)
		fmt.Printf("%s() failed: %s", funcName, errorMessage)
		return false
	}
	return true
}

func createOutputStream(width, height int, codecId int, outputFormatContext *avformat.Context) (*avformat.Stream, *avcodec.Context, error) {
	outputCodec := avcodec.AvcodecFindEncoder(codecId)
	if outputCodec == nil {
		errorMessage := fmt.Sprintf("Failed to find encoder for codec id %d", codecId)
		return nil, nil, errors.New(errorMessage)
	}
	outputStream := outputFormatContext.AvformatNewStream(nil)
	if outputStream == nil {
		return nil, nil, errors.New("Failed to allocate new output stream")
	}

	outputCodecContext := outputCodec.AvcodecAllocContext3()
	if outputCodecContext == nil {
		return nil, nil, errors.New("Failed to allocate output codec context")
	}

	outputCodecContext.SetThreadCount(outputCodecThreadCount)
	outputCodecContext.SetBitRate(outputCodecBitRate)
	outputCodecContext.SetWidth(width)
	outputCodecContext.SetHeight(height)
	timeBase := avutil.NewRational(1, outputStreamFrameRate)
	outputStream.SetTimeBase(timeBase)
	outputCodecContext.SetTimeBase(timeBase)
	outputCodecContext.SetGopSize(outputCodecGopSize)
	outputCodecContext.SetPixFmt(outputStreamPixFmt)
	if (outputFormatContext.Oformat().Flags() & avformat.AVFMT_FLAG_CUSTOM_IO) != 0 {
		outputCodecContext.SetFlags(outputCodecContext.Flags() | avcodec.AV_CODEC_FLAG_GLOBAL_HEADER)
	}

	return outputStream, outputCodecContext, nil

}

func main() {

	avlog.AvlogStartLoggingToFile("/home/alon/test_rtmp_listen.log")
	avlog.AvlogSetLevel(avlog.AV_LOG_DEBUG);

	avformat.AvformatNetworkInit()
	var inputFormatContext *avformat.Context
	var opts *avutil.Dictionary
	avutil.AvDictSet(&opts, "listen", "2", 0)
	//TODO: Set live to 1 in opts maybe?

	uri := "rtmp://127.0.0.1:1935/app/stream"
	fmt.Printf("Test rtmp server listening...\n")
	if errnum := avformat.AvformatOpenInput(&inputFormatContext, uri, nil, &opts); errnum < 0 {
		errorMessage := avutil.AvStrError(errnum)
		fmt.Printf("AvformatOpenInput() failed: %s\n", errorMessage)
		return
	}

	fmt.Printf("AvformatOpenInput() returned ok\n")

	fmt.Printf("Finding stream info\n")
	if errnum := inputFormatContext.AvformatFindStreamInfo(nil); errnum < 0 {
		errorMessage := avutil.AvStrError(errnum)
		fmt.Printf("AvformatFindStreamInfo() failed: %s\n", errorMessage)
		return
	}

	fmt.Printf("Dumping input format\n")
	inputFormatContext.AvDumpFormat(0, "", 0)

	inputStreams := inputFormatContext.Streams()
	inputCodecs := make([]*avcodec.Codec, len(inputStreams))
	inputCodecContexts := make([]*avcodec.Context, len(inputStreams))

	/* Initialize Input Codecs */
	for i, stream := range inputStreams {
		codecParams := stream.CodecPar()
		codecId := codecParams.CodecId()
		codec := avcodec.AvcodecFindDecoder(codecId)
		if codec == nil {
			fmt.Printf("Failed to find decoder for codec id %d\n", codecId)
			return
		}
		codecContext := codec.AvcodecAllocContext3()
		if codecContext == nil {
			fmt.Printf("Failed to allocate codec context\n")
			return
		}

		if error := avcodec.AvcodecParametersToContext(codecContext, codecParams); error < 0 {
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to get avcodec parameters: %s\n", errorMessage)
			return
		}

		if error := codecContext.AvcodecOpen2(codec, nil); error < 0 {
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to open input codec: %s\n", errorMessage)
			return
		}

		inputCodecs[i] = codec
		inputCodecContexts[i] = codecContext
	}

	/* Create output */
	avcodec.AvcodecRegisterAll()
	avformat.AvRegisterAll()
	outputFormat := avformat.AvGuessFormat(outputFormatName, "", "")
	if outputFormat == nil {
		fmt.Printf("Failed to guess format for output format '%s'\n", outputFormatName)
		return
	}
	outputFormatContext := avformat.AvformatAllocContext()
	if outputFormatContext == nil {
		fmt.Printf("Failed to allocate output format context\n")
		return
	}

//	outputStreams := make([]*avformat.Stream, len(inputStreams))
//	for _, inputStream := range(inputStreams) {
//		codecParams := inputStream.CodecPar()
//		codecId := codecParams.CodecId()
//	}

	fmt.Printf("Number of streams: %d\n", inputFormatContext.NbStreams())
	var inputPacket avcodec.Packet
//	var outputPacket avcodec.Packet
	frame := avutil.AvFrameAlloc()

	for {
		/* Get Frame */
		if error := inputFormatContext.AvReadFrame(&inputPacket); error < 0 {
			inputPacket.AvFreePacket()
			if error == avutil.AVERROR_EAGAIN {
				continue
			}
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to read frame: %s\n", errorMessage)
			break
		}
		streamIndex := inputPacket.StreamIndex()
		inputCodecContext := inputCodecContexts[streamIndex]
		if error := inputCodecContext.AvCodecSendPacket(&inputPacket); error < 0 {
			inputPacket.AvFreePacket()
			if error == avutil.AVERROR_EAGAIN {
				continue
			}
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to send packet to input codec: %s\n", errorMessage)
			break
		}

		if error := inputCodecContext.AvCodecReceiveFrame(frame); error < 0 {
			inputPacket.AvFreePacket()
			if error == avutil.AVERROR_EAGAIN {
				continue
			}
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to receive frame from input codec: %s\n", errorMessage)
			break
		}

		timeStamp := avutil.AvFrameGetBestEffortTimestamp(frame)
		frame.SetPts(timeStamp)

		/* Encode Frame */

		inputPacket.AvFreePacket()
	}

	avutil.AvFrameFree(frame)

}