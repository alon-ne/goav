package main

import (
	"fmt"
	"errors"
	"os"
	"github.com/giorgisio/goav/avlog"
	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avutil"
	"github.com/giorgisio/goav/avformat"
	log "github.com/cihub/seelog"

)

const (
	outputCodecThreadCount = 4
	outputCodecBitRate = 300000
	outputStreamFrameRate = 30
	outputCodecGopSize = 12
	outputStreamPixFmt = avcodec.AV_PIX_FMT_YUV420P
	outputFrameAlignment = 32

)
func CreateInputCodecContext(codecId avcodec.CodecId) (*avcodec.Context, error) {
	fmt.Printf("Calling AvcodecFindDecoder()\n")
	videoCodec := avcodec.AvcodecFindDecoder(codecId)
	if videoCodec == nil {
		return nil, errors.New("Failed to find decoder")
	}
	inputCodecContext := videoCodec.AvcodecAllocContext3()
	if inputCodecContext == nil {
		return nil, errors.New("Failed to allocate video codec context")
	}

	//TODO: Get width, height, and pixelFormat from rtmp metadata
	hardCodedWidth := 480
	hardCodedHeight := 360
	hardCodecPixelFormat := avcodec.PixelFormat(avcodec.AV_PIX_FMT_YUV420P)

	inputCodecContext.SetWidth(hardCodedWidth)
	inputCodecContext.SetHeight(hardCodedHeight)
	inputCodecContext.SetPixFmt(hardCodecPixelFormat)

	log.Debugf("Calling AvcodecOpen2()")
	if error := inputCodecContext.AvcodecOpen2(videoCodec, nil); error < 0 {
		//TODO: Deallocate avcodec context
		errorMessage := avutil.AvStrError(error)
		return nil, errors.New("Could not open input codec: " + errorMessage)
	}
	log.Debugf("AvcodecOpen2() returned ok")

	/*timeBase := p.videoCodecContext.TimeBase()
	if (timeBase.Num() > 1000) && (timeBase.Den() == 1) {
		timeBase.SetDen(1000)
	}*/

	return inputCodecContext, nil
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

func OpenVideo(outputStream *avformat.Stream, outputCodecContext *avcodec.Context) (error) {
	var opts *avutil.Dictionary
	avutil.AvDictSet(&opts, "crf", "20", 0);
	avutil.AvDictSet(&opts, "preset", "veryfast", 0);
	avutil.AvDictSet(&opts, "profile", "main", 0);
	avutil.AvDictSet(&opts, "level", "31", 0);
	avutil.AvDictSet(&opts, "x264opts", "keyint=30:bframes=2:min-keyint=30:ref=3:b-pyramid=0:b-adapt=0:no-scenecut", 0);

	log.Debugf("<3>")

	if errNum := outputCodecContext.AvcodecOpen2(nil, &opts); errNum < 0 {
		fmt.Printf("AvcodecOpen2 returend error %d\n", errNum)
		errorMessage := avutil.AvStrError(errNum)
		return errors.New("Failed to open output codec: " + errorMessage)
	}

	codecParams := outputStream.CodecPar()
	if errNum := avcodec.AvcodecParametersFromContext(codecParams, outputCodecContext); errNum < 0 {
		errorMessage := avutil.AvStrError(errNum)
		return errors.New("Failed to get codec parameters from context: " + errorMessage)
	}
	log.Debugf("Output video codec type: %d", outputCodecContext.CodecId)
	return nil
}

const 	maxArraySize = 1 << 31 - 1
type maxArray *[maxArraySize]uint8

func decodeFrame(data []byte, inputCodecContext *avcodec.Context) (*avutil.Frame, *avutil.Error) {
	var avpacket avcodec.Packet
	avpacket.AvInitPacket()
	packetSize := len(data)
	packetData := avutil.AvMalloc(uintptr(packetSize))
	defer avutil.AvFree(packetData)
	copy(maxArray(packetData)[:], data)
	avpacket.SetData(packetData)
	avpacket.SetSize(packetSize)

	log.Debugf("AvReadFrame() returned ok")
	if errNum := inputCodecContext.AvCodecSendPacket(&avpacket); errNum < 0 {
		averror := &avutil.Error{errNum}
		log.Debugf("AvCodecSendPacket returned error: %v", averror)
		return nil, averror
	}
	//TODO: Understand if we really need to re-allocate in every iteration
	frame := avutil.AvFrameAlloc()
	if frame == nil {
		log.Errorf("Failed to allocate frame")
		return nil, &avutil.Error{avutil.AVERROR_ENOMEM}
	}

	log.Debugf("Calling AvCodecReceiveFrame()")
	if errNum := inputCodecContext.AvCodecReceiveFrame(frame); errNum < 0 {
		log.Debugf("AvCodecReceiveFrame() returned error: %d", errNum)
		return frame, &avutil.Error{errNum}
	}
	log.Debugf("AvCodecReceiveFrame() returned ok")

	pts := avutil.AvFrameGetBestEffortTimestamp(frame)
	frame.SetPts(pts)
	return frame, nil
}

func encodeFrame(outputCodecContext *avcodec.Context, frame *avutil.Frame) (*avcodec.Packet, *avutil.Error) {
	if errnum := outputCodecContext.AvCodecSendFrame(frame); errnum < 0 {
		return nil, &avutil.Error{errnum}
	}
	var encodedPacket avcodec.Packet
	encodedPacket.AvInitPacket()
	if errnum := outputCodecContext.AvCodecReceivePacket(&encodedPacket); errnum < 0 {
		return nil, &avutil.Error{errnum}
	}
	//Alon: Got this timestamp trick from ffmpeg muxing.c example
	presentationTimeStamp := frame.Pts()
	decompressionTimeStamp := frame.Pts()
	encodedPacket.SetPts(presentationTimeStamp);
	encodedPacket.SetDts(decompressionTimeStamp);
	return &encodedPacket, nil
}

func main() {
	fmt.Printf("goav decode and encode sample\n")
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <inputFileName> <outputFileName.mp4>\n", os.Args[0])
		return
	}
	inputFileName := os.Args[1]
	outputFileName := os.Args[2]
	outputFormatName := "mp4"

	avlog.AvlogStartLoggingToFile("/tmp/sample.log")
	avlog.AvlogSetLevel(avlog.AV_LOG_DEBUG);
	inputCodecId := avcodec.CodecId(avcodec.AV_CODEC_ID_H264)

	inputCodecContext, error := CreateInputCodecContext(inputCodecId)
	if error != nil {
		log.Errorf("Failed to create decoding codec context: %v", error)
		return
	}

	outputFormat := avformat.AvGuessFormat(outputFormatName, "", "")
	if outputFormat == nil {
		log.Errorf("Failed to guess format for %s", outputFormatName)
		return
	}

	log.Debugf("<1>")
	outputFormatContext := avformat.AvformatAllocContext()
	if outputFormatContext == nil {
		log.Errorf("Failed to allocate output format context")
		return
	}
	outputFormatContext.SetOformat(outputFormat)
	outputFormatContext.SetFilename(outputFileName)
	outputCodecId := outputFormat.VideoCodec()
	if outputCodecId == avcodec.AV_CODEC_ID_NONE {
		log.Errorf("No video codec for output format")
		return
	}

	width := inputCodecContext.Width()
	height := inputCodecContext.Height()
	outputStream, outputCodecContext, error := createOutputStream(width, height, outputCodecId, outputFormatContext)
	if error != nil {
		log.Errorf("Failed to create output stream: %v", error)
		return
	}

	log.Debugf("<2>")
	if error = OpenVideo(outputStream, outputCodecContext); error != nil {
		log.Errorf("Failed to open video: %v", error)
		return
	}
	log.Debugf("Dumping output format")
	outputFormatContext.AvDumpFormat(0, outputFileName, 1)
	haveFile := ((outputFormatContext.Flags() & avformat.AVFMT_NOFILE) == 0)
	if (haveFile) {
		var outputAvioContext *avformat.AvIOContext
		if errnum := avformat.AvIOOpen(&outputAvioContext, outputFileName, avformat.AVIO_FLAG_WRITE); errnum < 0 {
			log.Errorf("Failed to open output file '%s': %v", outputFileName, avutil.Error{errnum})
			return
		}
		outputFormatContext.SetPb(outputAvioContext)
	}

	var opts *avformat.Dictionary
	/*
	av_dict_set(&opts, "use_template", "1", 0);
	av_dict_set(&opts, "init_seg_name", "init-file-stream$RepresentationID$.mp4",0);
	av_dict_set(&opts, "media_seg_name", "chunk-file-stream$RepresentationID$-$Number%05d$.m4s",0)
	*/
	if errnum := outputFormatContext.AvformatWriteHeader(&opts); errnum < 0 {
		log.Errorf("Failed to write output file header: %v", avutil.Error{errnum})
		return
	}

	inputFile, error := os.Open(inputFileName)
	if error != nil {
		log.Errorf("Failed to open input file %s: %s", inputFileName, error)
		return
	}
	defer inputFile.Close()

	var receivedFrames int
	bufferSize := 4096
	buffer := make([]byte, bufferSize)

	for {
		bytesRead, error := inputFile.Read(buffer)
		if error != nil {
			log.Errorf("Failed to read buffer from input file: %s", error)
			break
		}
		buffer := buffer[:bytesRead]
		decodedFrame, averror := decodeFrame(buffer, inputCodecContext)
		if averror != nil {
			if averror.Num == avutil.AVERROR_EAGAIN {
				continue
			}
			log.Errorf("Failed to decode frame: %v", averror)
			continue
		}
		framePacketSize := avutil.AvFrameGetPktSize(decodedFrame)
		log.Debugf("Decoded frame of size %d", framePacketSize)
		receivedFrames++
		//Encode frame with ffmpeg
		encodedPacket, averror := encodeFrame(outputCodecContext, decodedFrame)
		if averror != nil {
			if averror.Num == avutil.AVERROR_EAGAIN {
				continue
			}
			log.Errorf("Failed to encode frame: %v", averror)
			break
		}
		log.Debugf("Encoded frame to packet of size %d", encodedPacket.Size())

		//timestamp := int64(<-p.rtmpPacketTimestamps)

		//TODO: Use real timestamp from rtmp, maybe translate it properly with AvPacketRescaleTs()
		timestamp := int64(receivedFrames) * 427
		encodedPacket.SetDts(timestamp)

		/*
		inputTimeBase := (*inputFormatCotext.Streams()).TimeBase()
		outputTimeBase := (*outputFormatContext.Streams()).TimeBase()
		encodedPacket.AvPacketRescaleTs(inputTimeBase, outputTimeBase)
		*/

		log.Debugf("Calling AvInterleavedWriteFrame(), writing frame of %d bytes", encodedPacket.Size())
		if errnum := outputFormatContext.AvInterleavedWriteFrame(encodedPacket); errnum < 0 {
			errorMessage := avutil.AvStrError(errnum)
			log.Errorf("Failed to write frame: %s", errorMessage)
			encodedPacket.AvPacketUnref()
			break
		}
		log.Debugf("AvInterleavedWriteFrame() returned ok")
		encodedPacket.AvPacketUnref()
	}

	log.Debugf("Writing trailer")
	if errnum := outputFormatContext.AvWriteTrailer(); errnum < 0 {
		errorMessage := avutil.AvStrError(errnum)
		log.Errorf("Failed to write trailer to output file: %s", errorMessage)
	}
}