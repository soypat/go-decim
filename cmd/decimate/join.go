package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	sortByColumn   int
	joinOutputName string
	joinOutputDir  string
	deleteRepeats  bool
)

func init() {
	rootCmd.AddCommand(joinCmd)
	joinCmd.Flags().BoolVarP(&deleteRepeats, "delete-repeats", "x", false, "Deletes repeated values.")
	joinCmd.Flags().IntVar(&sortByColumn, "sort-column", 0, "Column to sort by. If 0 does not sort.")
	joinCmd.Flags().StringVarP(&inputSeparator, "delimiter", "d", ",", "Delimiter.")
	joinCmd.Flags().StringVarP(&joinOutputName, "output", "o", "joined.csv", "Output name of joined file.")
}

// joinCmd represents the join command
var joinCmd = &cobra.Command{
	Use:   "join [filenames...]",
	Short: "join .csv files in current directory into one without downsampling",
	Long: `join processes numerical data only. Files must have
same number of columns and each may or may not have a header
User may choose to sort values in ascending order using --sort-column flag.

join does NOT downsample or modify data unless delete-repeats flag is called.

	Example:

decimate -d , join -o new.csv --sort-column 3 *

Asterisk joins all files in directory. Columns start at 1.
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if strings.Contains(joinOutputName, string(filepath.Separator)) || strings.Contains(joinOutputName, "/") {
			joinOutputDir = filepath.Dir(joinOutputName)
			joinOutputName = discardPath(joinOutputName)
		}
		if inputSeparator == "\\t" || inputSeparator == "tab" {
			inputSeparator = "\t"
		}
		if len(inputSeparator) != 1 {
			return errors.New("Delimiter should be one character long!")
		}
		joinOutputName = replaceCutset(joinOutputName, badFilenameChar, "-")
		if joinOutputName == "" {
			joinOutputName = "joined.csv"
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if args[0] == "*" && len(args) == 1 {
			args = getAllCsvNames()
		}
		fmt.Println("join starting")
		if err := joiner(args); err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		fmt.Println("join finished")
	},
}

// delete repeated values helper struct
type floatColumnSet map[string]bool

func (s *floatColumnSet) attemptAdd(f []float64) bool {
	representation := strings.Join(floats2Strings(f), "")
	_, present := (*s)[representation]
	if present {
		return false
	}
	(*s)[representation] = true
	return true
}

type floatCsv struct {
	columns      []floatColumn
	header       []string
	columnSorter int
}

type floatColumn struct {
	data         []float64
	columnSorter int
}

type byColumn []floatColumn

func (a byColumn) Len() int      { return len(a) }
func (a byColumn) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byColumn) Less(i, j int) bool {
	return a[i].data[a[i].columnSorter-1] < a[j].data[a[j].columnSorter-1]
}

func joiner(args []string) error {
	var headers [][]string
	var NumberOfColumns int
	set := make(floatColumnSet)
	csvObj := floatCsv{columns: []floatColumn{}}
	for _, arg := range args {
		fi, err := os.Open(arg)
		if err != nil {
			return err
		}
		defer fi.Close()
		r := csv.NewReader(fi)
		r.Comma = rune(inputSeparator[0])
		records, err := r.ReadAll()
		if err != nil {
			return err
		}
		// save header if present and skip for data
		if _, err := strings2Floats(records[0]); err != nil {
			headers = append(headers, records[0])
			csvObj.header = records[0]
			records = records[1:]
		}
		for i, row := range records {
			floatcol, err := strings2Floats(row)
			// check if is repeated value if flag is up
			if deleteRepeats {
				allGood := set.attemptAdd(floatcol)
				if !allGood {
					continue
				}
			}
			if NumberOfColumns == 0 {
				NumberOfColumns = len(floatcol)
			} else if NumberOfColumns != len(floatcol) {
				return fmt.Errorf("different number of columns between files detected")
			}
			if err != nil {
				return fmt.Errorf("error line %d+/-1 of file %s. %s", i+1, arg, err)
			}
			csvObj.columns = append(csvObj.columns, floatColumn{
				data:         floatcol,
				columnSorter: sortByColumn,
			})
		}
	}
	fo, err := os.Create(filepath.Join(joinOutputDir, joinOutputName))
	if err != nil {
		return err
	}
	defer fo.Close()
	if sortByColumn > 0 {
		sort.Sort(byColumn(csvObj.columns))
	}
	w := csv.NewWriter(fo)
	if len(csvObj.header) > 0 {
		_ = w.Write(csvObj.header)
	}
	for i := range csvObj.columns {
		err = w.Write(floats2Strings(csvObj.columns[i].data))
		if err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}

func strings2Floats(S []string) ([]float64, error) {
	F := make([]float64, len(S))
	for i, s := range S {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		F[i] = f
	}
	return F, nil
}

func floats2Strings(F []float64) []string {
	S := make([]string, len(F))
	for i, f := range F {
		S[i] = fmt.Sprintf("%e", f)
	}
	return S
}

func getAllCsvNames() []string {
	var fileNames []string
	files, err := ioutil.ReadDir(".")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".csv") && file.Name() != joinOutputName {
			fileNames = append(fileNames, file.Name())
		}
	}
	return fileNames
}
