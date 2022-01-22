package decim

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strconv"
	"testing"
)

func TestSampler(t *testing.T) {
	fp, _ := os.Open("testdata/ch4.csv")
	rd := csv.NewReader(fp)
	rd.Read() // read header
	rec, err := rd.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	c := &CSVXYer{xField: 0, yField: 1, records: rec}
	s := NewSampler(c, 1)
	var xs, ys []float64
	var x, y float64
	for ; err == nil; x, y, err = s.Next() {
		xs = append(xs, x)
		ys = append(ys, y)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if len(xs) >= c.Len() || len(ys) >= c.Len() {
		t.Fatal("did not decimate succesfully")
	}
}

type CSVXYer struct {
	xField, yField int
	records        [][]string
}

func (c *CSVXYer) XY(i int) (x, y float64) {
	rec := c.records[i]
	x, err := strconv.ParseFloat(rec[c.xField], 64)
	if err != nil {
		panic(err)
	}
	y, err = strconv.ParseFloat(rec[c.yField], 64)
	if err != nil {
		panic(err)
	}
	return x, y
}

func (c *CSVXYer) Len() int { return len(c.records) }
