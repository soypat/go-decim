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
	s := &Sampler{
		tol:  tol,
		xyer: xyer,
	}
	s.Reset()
	// and also calculate initial permissible max angles line should be contained in.
	s.setStartAngleLims()
	return s
}

// Reset sets the sampler to initial value.
func (s *Sampler) Reset() {
	x, y := s.xyer.XY(0)
	s.idx = 1
	s.xPrev = x
	s.yPrev = y
	s.xPivot = x
	s.yPivot = y
}

func (s *Sampler) Next() (x, y float64, err error) {
	n := s.xyer.Len()
	for s.idx < n {
		x, y = s.xyer.XY(s.idx)
		s.idx++
		if s.idx == n {
			// Return last data without modification.
			return x, y, nil
		}
		dx, dy := x-s.xPivot, y-s.yPivot
		actualAngle := math.Atan2(dy, dx)
		if math.IsNaN(actualAngle) || math.IsInf(actualAngle, 0) {
			return 0, 0, errors.New("got infinity or NaN")
		}
		if !(actualAngle > s.angleMin) || !(actualAngle < s.angleMax) {
			// The angle of the line exceeded permissible angle range.
			if s.Interp {
				s.yPivot = s.yPivot + (s.xPrev-s.xPivot)*(math.Tan(s.angleMax)+math.Tan(s.angleMin))/2 // interpolator
			} else {
				s.yPivot = s.yPrev
			}
			s.xPivot, s.xPrev, s.yPrev = s.xPrev, x, y
			s.setStartAngleLims()
			return x, y, nil
		}
		// We update the angle limits based on new point.
		loangle, hiangle := math.Atan2(dy-s.tol, dx), math.Atan2(dy+s.tol, dx)
		s.angleMin = math.Max(loangle, s.angleMin)
		s.angleMax = math.Min(hiangle, s.angleMax)
		s.xPrev = x
		s.yPrev = y
	}
	return x, y, io.EOF
}

func (s *Sampler) setStartAngleLims() {
	if s.idx >= s.xyer.Len() {
		// Out of range. Stream is exhausted.
		return
	}
	x, y := s.xyer.XY(s.idx)
	dx, dy := x-s.xPivot, y-s.yPivot
	s.angleMax = math.Atan2(dy+s.tol, dx)
	s.angleMin = math.Atan2(dy-s.tol, dx)
}

// XYer processes xyer argument data and returns the downsampled data
func (s *Sampler) XYer() XYer {
	s.Reset()
	v := &sliceXYer{}
	var err error
	var x, y float64
	for {
		x, y, err = s.Next()
		if err != nil {
			break
		}
		v.x = append(v.x, x)
		v.y = append(v.y, y)
	}
	if !errors.Is(err, io.EOF) {
		panic(err)
	}
	return v
}

type sliceXYer struct {
	x, y []float64
}

func (s *sliceXYer) XY(i int) (x, y float64) {
	return s.x[i], s.y[i]
}

func (s *sliceXYer) Len() int {
	return len(s.x)
}
