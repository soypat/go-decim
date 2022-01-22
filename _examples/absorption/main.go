package main

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/soypat/go-decim"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
)

const (
	width  = font.Centimeter * 40
	height = font.Centimeter * 25
)

func main() {
	fp, _ := os.Open("../../testdata/ch4.csv")
	rd := csv.NewReader(fp)
	rec, err := rd.ReadAll()
	if err != nil {
		panic(err)
	}
	c := &CSVXYer{xField: 0, yField: 1, records: rec[1:]} // rec[1:] to skip header
	s := decim.NewSampler(c, 1e-4)
	pc := plot.New()
	cplot, _ := plotter.NewLine(c)
	pc.Add(cplot)
	pc.Save(width, height, "original.png")

	pn := plot.New()
	nplot, _ := plotter.NewLine(s.XYer())
	pn.Add(nplot)
	pn.Save(width, height, "decimated.png")
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
