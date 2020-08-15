package file

import (
	"bufio"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// WriteFileCSV takes in a 2D array and overwrites to a csv file
func WriteFileCSV(records [][]string, file string) error {
	err := os.Remove(file)
	if err != nil {
		return errors.Wrap(err, "file deletion failed")
	}

	createPath(file)
	csvFile, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "file creation failed")
	}
	defer csvFile.Close()

	w := csv.NewWriter(csvFile)
	if err := w.WriteAll(records); err != nil {
		return errors.Wrap(err, "error writing record")
	}

	return nil
}

// WriteLineCSV takes in a row and appends to a CSV file
func WriteLineCSV(record []string, file string) error {
	createPath(file)
	csvFile, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "file open failed")
	}
	defer csvFile.Close()

	w := csv.NewWriter(csvFile)
	if err := w.Write(record); err != nil {
		return errors.Wrap(err, "error writing record")
	}
	w.Flush()

	return nil
}

// UpdateLinesCSV takes in a row and updates multiple a CSV file by a column
// In case of nil row, the line is deleted
func UpdateLinesCSV(newRecord []string, file, value string, col int) error {
	lines, err := ReadCSV(file)
	if err != nil {
		return errors.Wrap(err, "file open failed")
	}

	records := lines
	for i, line := range lines {
		if value == line[col] {
			records[i] = newRecord
		}
	}

	WriteFileCSV(records, file)

	return nil
}

// UpdateColsCSV takes in a column and a value
// and updates the column across multiple rows
func UpdateColsCSV(newValue string, newCol int, queryVal string, queryCol int, file string) error {
	lines, err := ReadCSV(file)
	if err != nil {
		return errors.Wrap(err, "file open failed")
	}

	records := lines
	for i, line := range lines {
		if queryVal == line[queryCol] {
			records[i][newCol] = newValue
		}
	}

	WriteFileCSV(records, file)

	return nil
}

// ReadCSV reads the entire file into a 2D array
func ReadCSV(file string) ([][]string, error) {
	createPath(file)
	csvFile, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrap(err, "file open failed")
	}
	defer csvFile.Close()

	reader := csv.NewReader(bufio.NewReader(csvFile))
	var lines [][]string

	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		lines = append(lines, line)
	}

	return lines, nil
}

func createPath(file string) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		buff := strings.Split(file, "/")
		filedir := strings.Join(buff[:len(buff)-1], "/")
		os.MkdirAll(filedir, 0700)
	}
}
