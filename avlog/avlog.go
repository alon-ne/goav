package avlog

/*
#cgo pkg-config: libavutil
#include <libavutil/log.h>
#include <stdio.h>

FILE* logFile = NULL;

static inline void myLoggingFunction(void* unused, int level, const char* format, va_list arg)
{
	FILE* test = fopen("/home/alon/test2", "w");
	fclose(test);
	//TODO: Make this thread safe maybe?
	if (logFile == NULL)
	{
		FILE* couldNotOpenFile = fopen("/home/alon/could-not-open-av-log", "a");
		fprintf(couldNotOpenFile, ":(\n");
		fclose(couldNotOpenFile);
		return;
	}

	vfprintf(logFile, format, arg);
	fflush(logFile);
}

static inline void av_log_start_logging_to_file(const char* fileName)
{
	FILE* test = fopen("/home/alon/test1", "w");
	fclose(test);

	logFile = fopen(fileName, "a");
	fprintf(logFile, "----- goav log -----\n");
	fflush(logFile);
	av_log_set_callback(&myLoggingFunction);
}

static inline void av_log_stop_logging_to_file()
{
	fclose(logFile);
	av_log_set_callback(&av_log_default_callback);
}

*/
import "C"

const AV_LOG_DEBUG = 48

func AvlogSetLevel(level int) {
	C.av_log_set_level(C.int(level))
}

func AvlogGetLevel() int {
	return int(C.av_log_get_level())
}

func AvlogStartLoggingToFile(fileName string) {
	C.av_log_start_logging_to_file(C.CString(fileName))
}

func AvlogStopLoggingToFile() {
	C.av_log_stop_logging_to_file()
}