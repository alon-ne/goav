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

func initializeLog() {
	avlog.AvlogStartLoggingToFile("/home/alon/test_rtmp_listen.log")
	avlog.AvlogSetLevel(avlog.AV_LOG_DEBUG)
}

func createInputFormatContext() (*avformat.Context, error) {
	avformat.AvformatNetworkInit()
	var inputFormatContext *avformat.Context
	var opts *avutil.Dictionary
	avutil.AvDictSet(&opts, "listen", "1", 0)
	uri := "rtmp://127.0.0.1:1935/app/stream"
	fmt.Printf("Test rtmp server listening...\n")
	if errNum := avformat.AvformatOpenInput(&inputFormatContext, uri, nil, &opts); errNum < 0 {
		errorMessage := avutil.AvStrError(errNum)
		return nil, errors.New("AvformatOpenInput() failed: " + errorMessage)
	}

	fmt.Printf("AvformatOpenInput() returned ok\n")
	fmt.Printf("Finding stream info\n")
	if errNum := inputFormatContext.AvformatFindStreamInfo(nil); errNum < 0 {
		errorMessage := avutil.AvStrError(errNum)
		return nil, errors.New("AvformatFindStreamInfo() failed: " + errorMessage)
	}
	fmt.Printf("Dumping input format\n")
	inputFormatContext.AvDumpFormat(0, "", 0)
	return inputFormatContext, nil
}

func createOutputFormatContext() (*avformat.Context, error) {
	avcodec.AvcodecRegisterAll()
	avformat.AvRegisterAll()
	outputFormat := avformat.AvGuessFormat(outputFormatName, "", "")
	if outputFormat == nil {
		return nil, errors.New("Failed to guess format for output format " + outputFormatName)
	}

	outputFormatContext := avformat.AvformatAllocContext()
	if outputFormatContext == nil {
		return nil, errors.New("Failed to allocate output format context")
	}

	return outputFormatContext, nil
}

func createInputStreams(inputFormatContext *avformat.Context) ([]*avformat.Stream, []*avcodec.Context, error) {
	inputStreams := inputFormatContext.Streams()
	numInputStreams := len(inputStreams)
	inputCodecs := make([]*avcodec.Codec, numInputStreams)
	inputCodecContexts := make([]*avcodec.Context, len(inputStreams))
	for streamIndex, inputStream := range inputStreams {
		inputCodecParams := inputStream.CodecPar()
		inputStream.CodecPar()
		inputCodecId := inputCodecParams.CodecId()
		codec := avcodec.AvcodecFindDecoder(inputCodecId)
		if codec == nil {
			errorMessage := fmt.Sprintf("Failed to find decoder for codec id %d", inputCodecId)
			return nil, nil, errors.New(errorMessage)
		}
		inputCodecContext := codec.AvcodecAllocContext3()
		if inputCodecContext == nil {
			return nil, nil, errors.New("Failed to allocate codec context")
		}

		if error := avcodec.AvcodecParametersToContext(inputCodecContext, inputCodecParams); error < 0 {
			errorMessage := avutil.AvStrError(error)
			return nil, nil, errors.New("Failed to get avcodec parameters: " + errorMessage)
		}

		if error := inputCodecContext.AvcodecOpen2(codec, nil); error < 0 {
			errorMessage := avutil.AvStrError(error)
			return nil, nil, errors.New("Failed to open input codec: " + errorMessage)
		}

		inputCodecs[streamIndex] = codec
		inputCodecContexts[streamIndex] = inputCodecContext
	}
	fmt.Printf("Number of streams: %d\n", inputFormatContext.NbStreams())

	return inputStreams, inputCodecContexts, nil
}

func createOutputStreams(
	inputStreams []*avformat.Stream,
	inputCodecContexts []*avcodec.Context,
	outputFormatContext *avformat.Context) ([]*avformat.Stream, []*avcodec.Context, error) {

	numInputStreams := len(inputStreams)
	outputStreams := make([]*avformat.Stream, numInputStreams)
	outputCodecContexts := make([]*avcodec.Context, numInputStreams)
	for streamIndex, inputStream := range inputStreams {
		inputCodecContext := inputCodecContexts[streamIndex]
		inputCodecParams := inputStream.CodecPar()
		outputStream, outputCodecContext, error := createOutputStream(inputCodecParams, inputCodecContext, outputFormatContext)
		if error != nil {
			errorMessage := fmt.Sprintf("Failed to create output stream %d: %v", streamIndex, error)
			return nil, nil, errors.New(errorMessage)
		}
		outputStreams[streamIndex] = outputStream
		outputCodecContexts[streamIndex] = outputCodecContext
	}
	return outputStreams, outputCodecContexts, nil
}


func readFrame(inputFormatContext *avformat.Context, inputPacket *avcodec.Packet) (gotCompleteFrame bool, haveMoreFrames bool, error error) {
	gotCompleteFrame = true
	haveMoreFrames = true
	if errNum := inputFormatContext.AvReadFrame(inputPacket); errNum < 0 {
		inputPacket.AvPacketUnref()
		if errNum == avutil.AVERROR_EAGAIN {
			gotCompleteFrame = false
		} else if errNum == avutil.AVERROR_EOF {
			haveMoreFrames = false
		} else {
			avErrorMessage := avutil.AvStrError(errNum)
			errorMessage := fmt.Sprintf("Failed to read frame: %s (%d)\n", avErrorMessage, errNum)
			error = errors.New(errorMessage)
		}
	}
	return
}

func decodeFrame(
	inputPacket *avcodec.Packet,
	inputCodecContext *avcodec.Context,
	decodedFrame *avutil.Frame) (gotCompleteFrame bool, error error) {

	gotCompleteFrame = true
	if errNum := inputCodecContext.AvCodecSendPacket(&inputPacket); errNum < 0 {
		inputPacket.AvPacketUnref()
		if errNum == avutil.AVERROR_EAGAIN {
			gotCompleteFrame = false
		} else {
			avErrorMessage := avutil.AvStrError(errNum)
			errorMessage := "Failed to send packet to input codec: " + avErrorMessage
			error = errors.New(errorMessage)
		}
		return
	}

	if errNum := inputCodecContext.AvCodecReceiveFrame(decodedFrame); errNum < 0 {
		inputPacket.AvPacketUnref()
		if errNum == avutil.AVERROR_EAGAIN {
			gotCompleteFrame = false
		} else {
			avErrorMessage := avutil.AvStrError(errNum)
			errorMessage := "Failed to receive frame from input codec: " + avErrorMessage
			error = errors.New(errorMessage)
		}
		return
	}

	timeStamp := avutil.AvFrameGetBestEffortTimestamp(decodedFrame)
	decodedFrame.SetPts(timeStamp)
	return
}

func encodeFrame(
	outputCodecContext *avcodec.Context,
	inputStream *avformat.Stream,
	outputStream *avformat.Stream,
	inputFrame *avutil.Frame) (*avcodec.Packet, *avutil.Error) {
	if errNum := outputCodecContext.AvCodecSendFrame(inputFrame); errNum < 0 {
		fmt.Printf("AvCodecSendFrame() failed with error %d\n", errNum)
		return nil, &avutil.Error{errNum}
	}
	var encodedPacket avcodec.Packet
	encodedPacket.AvInitPacket()
	if errNum := outputCodecContext.AvCodecReceivePacket(&encodedPacket); errNum < 0 {
		fmt.Printf("AvCodecReceivePacket() failed with error %d\n", errNum)
		return nil, &avutil.Error{errNum}
	}

	presentationTimeStamp := inputFrame.Pts()
	decompressionTimeStamp := inputFrame.Pts()
	streamIndex := inputStream.Index()
	inputTimeBase := inputStream.TimeBase()
	outputTimeBase := outputStream.TimeBase()
	encodedPacket.SetStreamIndex(streamIndex)
	encodedPacket.SetPts(presentationTimeStamp)
	encodedPacket.SetDts(decompressionTimeStamp)
	encodedPacket.AvPacketRescaleTs(inputTimeBase, outputTimeBase)

	return &encodedPacket, nil
}

func createOutputStream(
	inputCodecParams *avcodec.CodecParameters,
	inputCodecContext *avcodec.Context,
	outputFormatContext *avformat.Context) (*avformat.Stream, *avcodec.Context, error) {

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
		errorMessage := fmt.Sprintf("Unknown codec type %d", codecType)
		return nil, nil, errors.New(errorMessage)

	}

	if error != nil {
		errorMessage := fmt.Sprintf("Failed to create output stream: %v", error)
		return nil, nil, errors.New(errorMessage)
	}

	return outputStream, outputCodecContext, nil
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
	if errNum := outputCodecContext.AvcodecOpen2(nil, nil); errNum < 0 {
		fmt.Printf("AvcodecOpen2() failed with error %d\n", errNum)
		return nil, nil, &avutil.Error{errNum}
	}

	// copy the stream parameters to the output stream
	if errNum := avcodec.AvcodecParametersFromContext(outputStream.CodecPar(), outputCodecContext); errNum < 0 {
		fmt.Printf("AvcodecParametersFromContext() failed with error %d\n", errNum)
		return nil, nil, &avutil.Error{errNum}
	}


	return outputStream, outputCodecContext, nil
}

func main() {
	initializeLog()

	inputFormatContext, error := createInputFormatContext()
	if error != nil {
		fmt.Printf("Failed to open input format context: %v\n", error)
		return
	}

	outputFormatContext, error := createOutputFormatContext()
	if error != nil {
		fmt.Printf("Failed to open output format context: %v\n", error)
		return
	}

	inputStreams, inputCodecContexts, error := createInputStreams(inputFormatContext)
	if error != nil {
		fmt.Printf("Failed to create input streams: %v\n", error)
		return
	}

	outputStreams, outputCodecContexts, error := createOutputStreams(inputStreams, inputCodecContexts, outputFormatContext)
	if error != nil {
		fmt.Printf("Failed to create output streams: %v\n", error)
		return
	}

	var inputPacket avcodec.Packet
	decodedFrame := avutil.AvFrameAlloc()
	numInputStreams := len(inputStreams)
	numFrames := make([]int, numInputStreams)

	for {
		gotCompleteFrame, haveMoreFrames, error := readFrame(inputFormatContext, &inputPacket)

		if error != nil {
			fmt.Printf("Failed to read frame: %v", error)
			return
		}

		if !haveMoreFrames {
			break
		}

		if !gotCompleteFrame {
			continue
		}

		streamIndex := inputPacket.StreamIndex()
		inputStream := inputStreams[streamIndex]
		outputStream := outputStreams[streamIndex]

		inputCodecContext := inputCodecContexts[streamIndex]
		gotCompleteFrame, error = decodeFrame(inputPacket, inputCodecContext, decodedFrame)
		if error != nil {
			fmt.Printf("Failed to decode frame: %v", error)
			return
		}

		if !gotCompleteFrame {
			continue
		}

		streamNumFrames := &numFrames[streamIndex]
		*streamNumFrames++
		fmt.Printf("Stream %d: Received %d frames\n", streamIndex, *streamNumFrames)

		/* Encode Frame */
		outputCodecContext := outputCodecContexts[streamIndex]
		fmt.Printf("Encoding frame to stream %d\n", streamIndex)
		encodedPacket, error := encodeFrame(outputCodecContext, inputStream, outputStream, decodedFrame)
		if error != nil {
			inputPacket.AvPacketUnref()
			fmt.Printf("Failed to encode frame: %v\n", error)
			return
		}

		/* Write frame */
		if error := outputFormatContext.AvInterleavedWriteFrame(encodedPacket); error < 0 {
			errorMessage := avutil.AvStrError(error)
			fmt.Printf("Failed to write frame: %s\n", errorMessage)
			inputPacket.AvPacketUnref()
			encodedPacket.AvPacketUnref()
			return
		}

		encodedPacket.AvPacketUnref()
		inputPacket.AvPacketUnref()

	}

	//avutil.AvFrameFree(frame)
}
