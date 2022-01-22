package decim

import (
	"errors"
	"io"
	"math"
)

// Same interface as gonum/plot XYer.
type XYer interface {
	XY(i int) (x, y float64)
	Len() int
}

type Sampler struct {
	idx                          int
	tol                          float64
	xPivot, yPivot, xPrev, yPrev float64
	angleMin, angleMax           float64
	xyer                         XYer
	// Interp attempts to lessen the error
	// by choosing next y value such that
	// the line is contained in the middle of
	// the angle limit range. Setting interp
	// means y values will not coincide with input data.
	Interp bool
}

func NewSampler(xyer XYer, tol float64) *Sampler {
	if xyer == nil {
		panic("got nil xyer")
	}
	if xyer.Len() < 3 {
		panic("need at least 3 points to downsample")
	}
	// We initialize x and y values
	x, y := xyer.XY(0)
	s := &Sampler{
		idx:   1,
		tol:   tol,
		xPrev: x, yPrev: y,
		xPivot: x, yPivot: y,
		xyer: xyer,
	}
	// and also calculate initial permissible max angles line should be contained in.
	s.setStartAngleLims()
	return s
}

func (a *Sampler) setStartAngleLims() {
	x, y := a.xyer.XY(a.idx)
	dx, dy := x-a.xPivot, y-a.yPivot
	a.angleMax = math.Atan2(dy+a.tol, dx)
	a.angleMin = math.Atan2(dy-a.tol, dx)
}

func (a *Sampler) Next() (x, y float64, err error) {
	n := a.xyer.Len()
	for a.idx < n {
		x, y = a.xyer.XY(a.idx)
		a.idx++
		dx, dy := x-a.xPivot, y-a.yPivot
		actualAngle := math.Atan2(dy, dx)
		if math.IsNaN(actualAngle) || math.IsInf(actualAngle, 0) {
			return 0, 0, errors.New("got infinity or NaN")
		}
		if !(actualAngle > a.angleMin) || !(actualAngle < a.angleMax) {
			// The angle of the line exceeded permissible angle range.
			if a.Interp {
				a.yPivot = a.yPivot + (a.xPrev-a.xPivot)*(math.Tan(a.angleMax)+math.Tan(a.angleMin))/2 // interpolator
			} else {
				a.yPivot = a.yPrev
			}
			a.xPivot, a.xPrev, a.yPrev = a.xPrev, x, y
			a.setStartAngleLims()
			return x, y, nil
		}
		// We update the angle limits based on new point.
		loangle, hiangle := math.Atan2(dy-a.tol, dx), math.Atan2(dy+a.tol, dx)
		a.angleMin = math.Max(loangle, a.angleMin)
		a.angleMax = math.Min(hiangle, a.angleMax)
		a.xPrev = x
		a.yPrev = y
	}
	return x, y, io.EOF
}
