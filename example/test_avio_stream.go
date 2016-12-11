package main

import (
	"fmt"
	"github.com/giorgisio/goav/avformat"
	"os"
	"github.com/giorgisio/goav/avutil"
	"github.com/giorgisio/goav/avlog"
)

var inputFile *os.File
var outputFile *os.File

type TestStream struct {
	name string
}

func (s *TestStream) String() string { return s.name }

func (s *TestStream) ReadPacket(buf avformat.AvIOPacket, bufSize int) int {
	fmt.Printf("go: ReadPacket: reading %d bytes\n", bufSize)
	slice := buf[0:bufSize]
	bytesRead, err := inputFile.Read(slice)
	if (err != nil) {
		fmt.Printf("Failed to read from input file: %s\n", err.Error())
		return -1
	}
	outputFile.Write(buf[0:bytesRead])
	return bytesRead
}

func (s *TestStream) WritePacket(buf avformat.AvIOPacket, bufSize int) int {
	return 0
}

func (s *TestStream) Seek(offset int64, whence int) int64 {
	return 0
}

func main() {
	avlog.AvlogStartLoggingToFile("/home/alon/goav.log")
	logLevel := avlog.AvlogGetLevel()
	avlog.AvlogSetLevel(avlog.AV_LOG_DEBUG);
	logLevel = avlog.AvlogGetLevel()
	fmt.Printf("Log level is %d\n", logLevel)

	bufferSize := 4096
	stream := TestStream{"TestStream"}
	fileName := "/home/alon/Downloads/ironman.mp4"
	fmt.Printf("Opening input file\n")
	var err error
	inputFile, err = os.Open(fileName)
	if (err != nil) {
		fmt.Printf("Failed to open input file %s: %s", fileName, err.Error())
		return
	}
	defer inputFile.Close()

	fmt.Printf("Opening output file\n")
	outputFile, err = os.Create("/home/alon/outFileGo")
	if (err != nil) {
		fmt.Printf("Failed to open output file: %s", err.Error())
		return
	}
	defer outputFile.Close()

	fmt.Printf("Allocating format context\n")
	formatContext := avformat.AvformatAllocContext()
	//TODO: Defer deallocating formatContext here
	formatContext.SetDebug(formatContext.Debug() | avformat.FF_FDEBUG_TS)

	fmt.Printf("Allocating AvIO context\n")
	avioContext, streamIndex, error := avformat.AvIOAllocContext(bufferSize, 0, &stream)
	if error != nil {
		fmt.Printf("Failed to allocate avio context: %s", error.Error())
	}
	defer avformat.AvIODeallocateContext(avioContext, streamIndex)

	formatContext.SetPb(avioContext)

	//Determine input format
	//probeBuffer := avutil.AvMalloc(uintptr(bufferSize))
	//defer avutil.AvFree(probeBuffer)

//	fmt.Printf("Reading from input stream\n")
//	stream.ReadPacket((avformat.AvIOPacket)(probeBuffer), bufferSize)
	//probeData := avformat.NewAvProbeData(unsafe.Pointer(probeBuffer), bufferSize, "")
	//probeData.
	//iFormat := avformat.AvProbeInputFormat(probeData, 1)
	//fmt.Printf("Setting Iformat\n")
	//formatContext.SetIformat(iFormat)
	//formatContext.SetFlags(avformat.AVFMT_FLAG_CUSTOM_IO)

/*	fmt.Printf("Finding input format\n")
	inputFormat := avformat.AvFindInputFormat("mp4")
*/

	fmt.Printf("Opening input\n")
	if rc := avformat.AvformatOpenInput(&formatContext, "", nil, nil); rc != 0 {
		errorMessage := avutil.AvStrError(rc)
		fmt.Printf("Failed to open input: %d, %s\n", rc, errorMessage)
		return
	}

	fmt.Printf("Finding stream info\n")
	if rc := formatContext.AvformatFindStreamInfo(nil); rc != 0 {
		fmt.Printf("Failed to find stream info: %d\n", rc)
		return
	}

	//TODO: defer avformat.AvFormatCloseInput()
	fmt.Printf("Dumping format\n")
	formatContext.AvDumpFormat(0, "", 0)
}
