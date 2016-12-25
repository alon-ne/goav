package avutil

const (
	AVERROR_EAGAIN = -11
	AVERROR_ENOMEM = -12

)

type Error struct {
	Num int
}

func (e *Error) Error() string {
	errorMessage := AvStrError(e.Num)
	return errorMessage
}