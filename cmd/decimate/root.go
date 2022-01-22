package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// flags
var tolerance float64 = 0.1 // default for tests
var xFlag, yFlag, inputSeparator, outputName, outputDir, outputExtension, floatFormat string
var interp, enforceComma, silent, noHeader bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "decimate",
	Short: "Reduce # of points of signals and curves.",
	Long: `Decimator is a tool for downsampling
(also known as decimating) numerical data.
Generates decimated files from a token separated
file for use in plotting tools. Column numbering
starts at 1.

Examples:

	decimate -x time -y "x,y,z" myFile.csv

Operates on the time x-column and 3 y-columns
named 'x', 'y' and 'z'.

	decimate -x time -y "*" -d tab aFile.tsv

Operates on the time x-column and all y-columns of a
tab separated file. 
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := checkParameters(args); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(args); err != nil {
			fmt.Printf("[ERR] %s", err)
			os.Exit(1)
		}
	},
}

type job struct {
	*csv.Writer
	xname, yname string
	tolerance    float64
	stepper
}

const badFilenameChar = "/\\:*?\"><|"

func getJobName(j job) string {
	sanitizedYname := replaceCutset(j.yname, badFilenameChar, "-")
	return filepath.Join(outputDir, outputName+"-"+sanitizedYname+"."+outputExtension)
}

func run(args []string) error {
	fi, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("error opening file")
	}
	rdr := csv.NewReader(fi)
	rdr.Comma = rune(inputSeparator[0])
	rdr.TrimLeadingSpace = true
	headers, err := rdr.Read()
	if err != nil {
		return err
	}
	if findNumerical(headers) >= 0 {
		return fmt.Errorf("numerical header entry found: %s", headers[findNumerical(headers)])
	}
	var yColNames []string
	if yColNames, err = parseHeader(headers); err != nil {
		return err
	}

	var yxIdx []int
	for _, v := range append(yColNames, xFlag) {
		i := findStringInSlice(v, headers)
		if i < 0 {
			return fmt.Errorf("%s is not in columns:\n%v", v, headers)
		}
		yxIdx = append(yxIdx, i)
	}
	// we have as many files to create as y columns given
	var jobs []*job
	for i := 0; i < len(yColNames); i++ {
		var algorithm stepper
		if interp {
			algorithm = interpStepper{}
		} else {
			algorithm = inPlaceStepper{}
		}
		j := job{
			xname:     xFlag,
			yname:     yColNames[i],
			tolerance: tolerance,
			stepper:   algorithm,
		}
		fo, err := os.Create(getJobName(j))
		defer fo.Close()
		if err != nil {
			return err
		}
		j.Writer = csv.NewWriter(fo)
		defer j.Flush()
		if !enforceComma {
			j.Writer.Comma = rune(inputSeparator[0])
		}
		if !noHeader {
			err = j.Write([]string{j.xname, j.yname})
		}
		if err != nil {
			return err
		}
		alert("creating file %s", getJobName(j))
		jobs = append(jobs, &j)
	}
	// begin doing the heavy lifting
	var EOF bool
	for !EOF {
		record, err := rdr.Read()
		if err != nil {
			if err.Error() == "EOF" {
				alert("finished writing files")
				EOF = true
				record = NaNslice(len(headers))
			} else {
				return err
			}
		}
		x, err := strconv.ParseFloat(record[yxIdx[len(yxIdx)-1]], 64)
		if err != nil {
			return err
		}
		for i := 0; i < len(yColNames); i++ {
			y, err := strconv.ParseFloat(record[yxIdx[i]], 64)
			if err != nil {
				return err
			}
			jobs[i].stepper = jobs[i].step(x, y)
			if jobs[i].stepper.ready() {
				if err := jobs[i].Write(jobs[i].stepper.values(floatFormat)); err != nil {
					panic(err)
				}
			}
		}
	}

	return nil
}

func parseHeader(headers []string) ([]string, error) {
	yColsSplit := splitColumns(yFlag)
	// Column number replacer
	if colNum, err := strconv.Atoi(xFlag); err == nil && colNum > 0 {
		if colNum > len(headers) || colNum == 0 {
			return nil, fmt.Errorf("x column number %d too large or zero. Have %d headers", colNum, len(headers))
		}
		xFlag = headers[colNum-1]
	}
	for _, h := range headers {
		if h != xFlag {
			for i, y := range yColsSplit {
				colNum, err := strconv.Atoi(y)
				if err == nil && colNum > 0 {
					if colNum > len(headers) || colNum == 0 {
						return nil, fmt.Errorf("y column number %d too large or zero. Have %d headers", colNum, len(headers))
					}
					yColsSplit[i] = headers[colNum-1]
				}
			}
		}
	}
	// if we are looking for all columns, "*" as -y flag
	if yColsSplit[0] == "*" && len(yColsSplit) == 1 {
		yColsSplit = []string{}
		for _, h := range headers {
			if h != xFlag {
				yColsSplit = append(yColsSplit, h)
			}
		}
	}
	return yColsSplit, nil
}

func NaNslice(n int) (nans []string) {
	for i := 0; i < n; i++ {
		nans = append(nans, "NaN")
	}
	return nans
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// actually modifies flag values!
func checkParameters(args []string) error {
	// filename
	if len(args) != 1 {
		fmt.Printf("args: %v", args)
		return errors.New("requires exactly one argument as input filename")
	}
	if _, err := os.Stat(args[0]); err != nil {
		return fmt.Errorf("opening %s. %s", args[0], err)
	}
	// y columns
	ycols := splitColumns(yFlag)
	if len(ycols) < 1 || yFlag == "" {
		return errors.New("found no y-column flag value")
	} else if findStringInSlice(xFlag, ycols) >= 0 {
		return errors.New("found x-column name/number within y-column values")
	}
	// Delimiters
	if strings.TrimSuffix(inputSeparator, "s") == "tab" || inputSeparator == "\\t" {
		yFlag = strings.ReplaceAll(yFlag, "\\t", "\t")
		inputSeparator = "\t"
	}
	if len(inputSeparator) != 1 {
		return errors.New("delimiter should be one character. '\\t' and 'tab' work as an option")
	}
	var iname string
	if strings.Contains(outputName, string(filepath.Separator)) || strings.Contains(outputName, "/") {
		outputDir = filepath.Dir(outputName)
		outputName = discardPath(outputName)
	}
	outputName, outputExtension = splitFileExtension(outputName)
	if outputExtension == "" || outputExtension == "<inputExtension>" {
		iname, outputExtension = splitFileExtension(discardPath(args[0]))
		if outputName == "<inputName>-<ycol>" || outputName == "" {
			outputName = iname
		}
	}
	// formatter
	const floatNum = .125
	if _, err := strconv.ParseFloat(fmt.Sprintf(floatFormat, floatNum), 64); err != nil {
		return errors.New("formatting option yielded error. example of usage: \n'%0.2f' for two decimal placed\n'%e' for scientific notation\nError: " + err.Error())
	}
	return nil
}

func init() {
	rootCmd.Flags().StringVarP(&outputName, "output", "o", "", "Output filename. Named after ycolumn. Extension by default is input file's")
	rootCmd.Flags().StringVarP(&floatFormat, "fformat", "f", "%.6e", "Floating point format")
	rootCmd.Flags().BoolVarP(&enforceComma, "comma", "c", false, "Force output to use comma as delimiter")
	rootCmd.Flags().StringVarP(&inputSeparator, "delimiter", "d", ",", "Delimiter token. Examples: '-d \\t' or '-d=\";\"'")
	rootCmd.Flags().Float64VarP(&tolerance, "tolerance", "t", 0.1, "Downsampling y-value tolerance.")
	rootCmd.Flags().StringVarP(&yFlag, "ycols", "y", "", "Y column names/numbers separated by delimiter. Numbering starts at 1. Pass -y=\"*\" to process all columns (required)")
	_ = rootCmd.MarkFlagRequired("ycols")
	rootCmd.Flags().StringVarP(&xFlag, "xcol", "x", "", "X column name. May be column number starting at 1. (required)")
	_ = rootCmd.MarkFlagRequired("xcol")
	rootCmd.Flags().BoolVarP(&interp, "interp", "i", false, "Use more aggressive interpolating algorithm. Changes y values")
	rootCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Silent execution (no printing).")
	rootCmd.Flags().BoolVarP(&noHeader, "headerless", "n", false, "If set does not print headers in new file.")
}

// returns -1 if string not found.
// else returns index in slice
func findStringInSlice(s string, sli []string) int {
	for i, v := range sli {
		if v == s {
			return i
		}
	}
	return -1
}

func splitColumns(y string) []string {
	s := strings.Split(yFlag, inputSeparator)
	return s
}

func discardPath(fname string) string {
	pathIndex := strings.LastIndex(fname, "/")
	if pathIndex != -1 && pathIndex < len(fname)-1 {
		fname = fname[1+pathIndex:]
	}
	return fname
}

func splitFileExtension(fname string) (string, string) {
	fileTypeIndex := strings.LastIndex(fname, ".")
	if fileTypeIndex == -1 {
		return fname, ""
	}
	return fname[:fileTypeIndex], fname[fileTypeIndex+1:]
}

func alert(format string, args ...interface{}) {
	if !silent {
		msg := fmt.Sprintf(format, args...)
		if args == nil {
			msg = fmt.Sprintf(format)
		}
		fmt.Print("[INFO] ", msg, "\n")
	}
}

func findNumerical(sli []string) int {
	for i, v := range sli {
		if isNumerical(v) {
			return i
		}
	}
	return -1
}

func isNumerical(s string) bool {
	_, err := strconv.ParseFloat(s, 32)
	return err == nil
}

func replaceCutset(str, oldcut, new string) string {
	oldies := make(map[rune]bool, len(oldcut))
	for _, r := range oldcut {
		oldies[r] = true
	}
	var newString string
	for _, v := range str {
		_, present := oldies[v]
		if present {
			newString += new
		} else {
			newString += string(v)
		}
	}
	return newString
}
