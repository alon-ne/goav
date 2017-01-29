package main

import (
	"errors"
	"fmt"
	"github.com/alon-ne/goav/avutil"
	"github.com/alon-ne/goav/avcodec"
	"github.com/alon-ne/goav/avformat"
	"github.com/alon-ne/goav/avlog"
)

const (
	outputAudioCodecId = avcodec.AV_CODEC_ID_AAC
	outputVideoCodecId = avcodec.AV_CODEC_ID_H264
	outputFormatName = "mp4"
	outputCodecThreadCount = 4
	outputCodecBitRate = 300000
	outputStreamFrameRate = 30
	outputCodecGopSize = 12
	outputStreamPixFmt = avcodec.AV_PIX_FMT_YUV420P
	outputFrameAlignment = 32

	outputSampleRate = 48000
)

func checkAvRet(funcName string, avfunc func() int) bool {
	if errnum := avfunc(); errnum < 0 {
		errorMessage := avutil.AvStrError(errnum)
		fmt.Printf("%s() failed: %s", funcName, errorMessage)
		return false
	}
	return true
}

func createVideoOutputStream(width, height int, outputCodecId int, outputFormatContext *avformat.Context) (*avformat.Stream, *avcodec.Context, error) {
	outputCodec := avcodec.AvcodecFindEncoder(outputCodecId)
	if outputCodec == nil {
		errorMessage := fmt.Sprintf("Failed to find encoder for codec id %d", outputCodecId)
		return nil, nil, errors.New(errorMessage)
	}

	//TODO: Maybe pass outputCodec here, see if that works
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
	if (outputFormatContext.Oformat().Flags() & avformat.AVFMT_GLOBALHEADER) != 0 {
		outputCodecContext.SetFlags(outputCodecContext.Flags() | avcodec.AV_CODEC_FLAG_GLOBAL_HEADER)
	}

	var opts *avutil.Dictionary
	avutil.AvDictSet(&opts, "crf", "20", 0)
	avutil.AvDictSet(&opts, "preset", "veryfast", 0)
	avutil.AvDictSet(&opts, "profile", "main", 0)
	avutil.AvDictSet(&opts, "level", "31", 0)
	avutil.AvDictSet(&opts, "x264opts", "keyint=30:bframes=2:min-keyint=30:ref=3:b-pyramid=0:b-adapt=0:no-scenecut", 0)

	if errNum := outputCodecContext.AvcodecOpen2(nil, &opts); errNum < 0 {
		fmt.Printf("AvcodecOpen2 returend error %d\n", errNum)
		errorMessage := avutil.AvStrError(errNum)
		return nil, nil, errors.New("Failed to open output codec: " + errorMessage)
	}

	codecParams := outputStream.CodecPar()
	if errNum := avcodec.AvcodecParametersFromContext(codecParams, outputCodecContext); errNum < 0 {
		errorMessage := avutil.AvStrError(errNum)
		return nil, nil, errors.New("Failed to get codec parameters from context: " + errorMessage)
	}

	return outputStream, outputCodecContext, nil
}

func createAudioOutputStream(sampleRate int, outputCodecId int, outputFormatContext *avformat.Context) (*avformat.Stream, *avcodec.Context, error) {
	outputCodec := avcodec.AvcodecFindEncoder(outputCodecId)
	if outputCodec == nil {
		errorMessage := fmt.Sprintf("Failed to find encoder for codec id %d", outputCodecId)
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

	/* set sample parameters */
	outputCodecContext.SetSampleRate(sampleRate);
	outputCodecContext.SetChannelLayout(avutil.AV_CH_LAYOUT_STEREO);
	outputCodecContext.SetChannels(2);
	outputCodecContext.SetBitRate(192000);
	outputCodecContext.SetSampleFmt(avutil.AV_SAMPLE_FMT_S16);

	timeBase := avutil.NewRational(1, outputStreamFrameRate)
	outputStream.SetTimeBase(timeBase)

	// some formats want stream headers to be separate
	if (outputFormatContext.Oformat().Flags() & avformat.AVFMT_GLOBALHEADER) != 0 {
		outputCodecContext.SetFlags(outputCodecContext.Flags() | avcodec.AV_CODEC_FLAG_GLOBAL_HEADER)
	}

	//TODO: Try passing outputCodec here, see if that works
	if errnum := outputCodecContext.AvcodecOpen2(nil, nil); errnum < 0 {
		fmt.Printf("AvcodecOpen2() failed with error %d\n", errnum)
		return nil, nil, &avutil.Error{errnum}
	}

	// copy the stream parameters to the output stream
	if errnum := avcodec.AvcodecParametersFromContext(outputStream.CodecPar(), outputCodecContext); errnum < 0 {
		fmt.Printf("AvcodecParametersFromContext() failed with error %d\n", errnum)
		return nil, nil, &avutil.Error{errnum}
	}


	return outputStream, outputCodecContext, nil
}

func encodeFrame(outputCodecContext *avcodec.Context, inputFrame *avutil.Frame) (*avcodec.Packet, *avutil.Error) {
	if errnum := outputCodecContext.AvCodecSendFrame(inputFrame); errnum < 0 {
		fmt.Printf("AvCodecSendFrame() failed with error %d\n", errnum)
		return nil, &avutil.Error{errnum}
	}
	var encodedPacket avcodec.Packet
	encodedPacket.AvInitPacket()
	if errnum := outputCodecContext.AvCodecReceivePacket(&encodedPacket); errnum < 0 {
		fmt.Printf("AvCodecReceivePacket() failed with error %d\n", errnum)
		return nil, &avutil.Error{errnum}
	}
	//Alon: Got this timestamp trick from ffmpeg muxing.c example
	presentationTimeStamp := inputFrame.Pts()
	decompressionTimeStamp := inputFrame.Pts()
	encodedPacket.SetPts(presentationTimeStamp)
	encodedPacket.SetDts(decompressionTimeStamp)
	return &encodedPacket, nil
}

func GetOutputCodecId(codecType avcodec.MediaType) int {
	outputAudioCodecId := avcodec.AV_CODEC_ID_AAC
	outputVideoCodecId := avcodec.AV_CODEC_ID_H264

	var outputCodecId int
	switch codecType {
	case avutil.AVMEDIA_TYPE_AUDIO:
		outputCodecId = outputAudioCodecId
	case avutil.AVMEDIA_TYPE_VIDEO:
		outputCodecId = outputVideoCodecId
	default:
		outputCodecId = avcodec.AV_CODEC_ID_FIRST_UNKNOWN
	}

	return outputCodecId
}

func main() {
	avlog.AvlogStartLoggingToFile("/home/alon/test_rtmp_listen.log")
	avlog.AvlogSetLevel(avlog.AV_LOG_DEBUG);

	avformat.AvformatNetworkInit()
	var inputFormatContext *avformat.Context
	var opts *avutil.Dictionary
	avutil.AvDictSet(&opts, "listen", "1", 0)
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
	numInputStreams := len(inputStreams)
	inputCodecs := make([]*avcodec.Codec, numInputStreams)
	inputCodecContexts := make([]*avcodec.Context, len(inputStreams))

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

	outputStreams := make([]*avformat.Stream, numInputStreams)
	outputCodecContexts := make([]*avcodec.Context, numInputStreams)

	/* Initialize Input and Output codecs and streams */
	for streamIndex, inputStream := range inputStreams {
		inputCodecParams := inputStream.CodecPar()
		inputStream.CodecPar()
		inputCodecId := inputCodecParams.CodecId()
		codec := avcodec.AvcodecFindDecoder(inputCodecId)
		if codec == nil {
			fmt.Printf("Failed to find decoder for codec id %d\n", inputCodecId)
			return
		}
		inputCodecContext := codec.AvcodecAllocContext3()
		if inputCodecContext == nil {
			fmt.Printf("Failed to allocate codec context\n")
			return
		}

		if error := avcodec.AvcodecParametersToContext(inputCodecContext, inputCodecParams); error < 0 {
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to get avcodec parameters: %s\n", errorMessage)
			return
		}

		if error := inputCodecContext.AvcodecOpen2(codec, nil); error < 0 {
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to open input codec: %s\n", errorMessage)
			return
		}

		inputCodecs[streamIndex] = codec
		inputCodecContexts[streamIndex] = inputCodecContext

		codecType := inputCodecParams.CodecType()
		var outputStream *avformat.Stream
		var outputCodecContext *avcodec.Context
		var error error

		switch codecType {
		case avutil.AVMEDIA_TYPE_AUDIO:
			width := inputCodecContext.Width()
			height := inputCodecContext.Height()
			outputStream, outputCodecContext, error = createVideoOutputStream(
				width,
				height,
				outputVideoCodecId,
				outputFormatContext)

		case avutil.AVMEDIA_TYPE_VIDEO:
			outputStream, outputCodecContext, error = createAudioOutputStream(
				outputSampleRate,
				outputAudioCodecId,
				outputFormatContext)

		default:
			fmt.Printf("Unknown codec type %d\n", codecType)
			return
		}

		if error != nil {
			fmt.Printf("Failed to create output stream: %v\n", error)
			return
		}
		outputStreams[streamIndex] = outputStream
		outputCodecContexts[streamIndex] = outputCodecContext
	}

	fmt.Printf("Number of streams: %d\n", inputFormatContext.NbStreams())
	var inputPacket avcodec.Packet
	decodedFrame := avutil.AvFrameAlloc()
	numFrames := make([]int, numInputStreams)

	for {
		/* Read Frame */
		if error := inputFormatContext.AvReadFrame(&inputPacket); error < 0 {
			inputPacket.AvPacketUnref()
			if error == avutil.AVERROR_EAGAIN {
				continue
			} else if error == avutil.AVERROR_EOF {
				fmt.Printf("Received EOF, stopping.")
				break
			}

			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to read frame: %s (%d)\n", errorMessage, error)
			break
		}

		/* Decode frame */
		streamIndex := inputPacket.StreamIndex()
		inputCodecContext := inputCodecContexts[streamIndex]
		if error := inputCodecContext.AvCodecSendPacket(&inputPacket); error < 0 {
			inputPacket.AvPacketUnref()
			if error == avutil.AVERROR_EAGAIN {
				continue
			}
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to send packet to input codec: %s\n", errorMessage)
			break
		}

		if error := inputCodecContext.AvCodecReceiveFrame(decodedFrame); error < 0 {
			inputPacket.AvPacketUnref()
			if error == avutil.AVERROR_EAGAIN {
				continue
			}
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to receive frame from input codec: %s\n", errorMessage)
			break
		}

		timeStamp := avutil.AvFrameGetBestEffortTimestamp(decodedFrame)
		decodedFrame.SetPts(timeStamp)
		streamNumFrames := &numFrames[streamIndex]
		*streamNumFrames++
		fmt.Printf("Stream %d: Received %d frames\n", streamIndex, *streamNumFrames)

		/* Encode Frame */
		outputCodecContext := outputCodecContexts[streamIndex]
		fmt.Printf("Encoding frame to stream %d\n", streamIndex)
		encodedPacket, error := encodeFrame(outputCodecContext, decodedFrame)
		if error != nil {
			fmt.Printf("Failed to encode frame: %v\n", error)
			return
		}

		inputStream := inputStreams[streamIndex]
		outputStream := outputStreams[streamIndex]
		encodedPacket.AvPacketRescaleTs(inputStream.TimeBase(), outputStream.TimeBase())
		encodedPacket.SetStreamIndex(streamIndex)

		/* Write frame */
		if error := outputFormatContext.AvInterleavedWriteFrame(encodedPacket); error < 0 {
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to write frame: %s\n", errorMessage)
			encodedPacket.AvPacketUnref()
			return
		}
		encodedPacket.AvPacketUnref()
		inputPacket.AvPacketUnref()

	}

	//avutil.AvFrameFree(frame)

}
