package decim

import (
	"errors"
	"io"
)

// Same interface as gonum/plot XYer.
type XYZer interface {
	XYZ(i int) (x, y, z float64)
	Len() int
}

type SpatialSampler struct {
	idx int
	tol float64
	// point in space representing origin of decimation in progress.
	pivot vec
	xyzer XYZer
}

func NewSpatialSampler(xyzer XYZer, tol float64) *SpatialSampler {
	if xyzer == nil {
		panic("got nil xyer")
	}
	if xyzer.Len() < 3 {
		panic("need at least 3 points to downsample")
	}
	// We initialize x and y values
	s := &SpatialSampler{
		tol:   tol,
		xyzer: xyzer,
	}
	s.Reset()
	return s
}

// Reset sets the sampler to initial value.
func (s *SpatialSampler) Reset() {
	s.pivot.x, s.pivot.y, s.pivot.z = s.xyzer.XYZ(0)
	s.idx = 1
}

func (s *SpatialSampler) Next() (x, y, z float64, err error) {
	n := s.xyzer.Len()
	_ = n
	return x, y, z, io.EOF
}

// XYer processes xyer argument data and returns the downsampled data
func (s *SpatialSampler) XYZer() XYZer {
	s.Reset()
	v := &sliceXYZer{}
	var err error
	var x, y, z float64
	for {
		x, y, z, err = s.Next()
		if err != nil {
			break
		}
		v.x = append(v.x, x)
		v.y = append(v.y, y)
		v.z = append(v.z, z)
	}
	if !errors.Is(err, io.EOF) {
		panic(err)
	}
	return v
}

type sliceXYZer struct {
	x, y, z []float64
}

func (s *sliceXYZer) XYZ(i int) (x, y, z float64) {
	return s.x[i], s.y[i], s.z[i]
}

func (s *sliceXYZer) Len() int {
	return len(s.x)
}
