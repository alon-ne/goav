package avutil


//#cgo pkg-config: libavcodec
//#include <libavcodec/avcodec.h>
import "C"
import (
	"unsafe"
)

func AvDictSet(d **Dictionary, key, value string, flags int) int {
	return int(C.av_dict_set(unsafe.Pointer(d), C.CString(key), C.CString(value), C.int(flags)))
}

func AvDictGet(d *Dictionary, key string, prev *DictionaryEntry, flags int) *DictionaryEntry {
	return (*DictionaryEntry)(C.av_dict_get(unsafe.Pointer(d), C.CString(key), unsafe.Pointer(prev), C.int(flags)))
}

func (e *DictionaryEntry) Key() string {
	return C.GoString(e.key)
}

func (e *DictionaryEntry) Value() string {
	return C.GoString(e.value)
}