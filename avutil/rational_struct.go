package avutil

//#include <libavutil/rational.h>
import "C"

func (r *Rational) Num() int {
	return int(r.num)
}

func (r *Rational) SetNum(num int) {
	r.num = C.int(num)
}

func (r *Rational) Den() int {
	return int(r.den)
}

func (r *Rational) SetDen(den int) {
	r.den = C.int(den)
}

func NewRational(num, den int) Rational {
	var r Rational
	r.SetNum(num)
	r.SetDen(den)
	return r
}
