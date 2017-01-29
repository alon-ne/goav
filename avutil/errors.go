package avutil

const (
	AVERROR_EAGAIN = -11
	AVERROR_ENOMEM = -12
	AVERROR_EOF = -541478725
)

type Error struct {
	Num int
}

func (e *Error) Error() string {
	errorMessage := AvStrError(e.Num)
	return errorMessage
}