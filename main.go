// csvsplit splits a CSV from STDIN, never splitting a row across files.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	colSep = ','
	rowSep = '\n'
	quote  = '"'
)

// QuotedCSVLineSplit is a SplitFunc for bufio.Scanner, which splits CSV rows.
func QuotedCSVLineSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	type State int
	const (
		Start State = iota
		UnquotedField
		QuotedField
		Quote
	)

	state := Start
	for i, b := range data {
		switch state {
		case Start:
			switch b {
			case quote:
				state = QuotedField
			case colSep:
				state = Start
			default:
				state = UnquotedField
			}
		case UnquotedField:
			switch b {
			case rowSep:
				return i + 1, data[0:i], nil
			case colSep:
				state = Start
			}
		case QuotedField:
			if b == quote {
				state = Quote
			}
		case Quote:
			switch b {
			case quote:
				// Just an escaped quote.
				state = QuotedField
			case colSep:
				// That was the end of the quoted field.
				state = Start
			case rowSep:
				return i + 1, data[0:i], nil
			default:
				// Invalid.
				return 0, nil, fmt.Errorf("invalid character following \" in quoted field: %s", string(b))
			}
		}
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

const initBufSizeBytes = 64 * 1024
const maxBufSizeBytes = 10 * 1024 * 1024

// Split splits a stream of CSV data every maxBytesPerFile.
// It requests a new output target from genNextFile for each split.
func Split(in io.Reader, maxBytesPerFile int, genNextFile func() (io.Writer, error)) error {
	var w *bufio.Writer
	currFileBytes := maxBytesPerFile // This forces a new file to be generated initially.

	scanner := bufio.NewScanner(in)

	scanner.Split(QuotedCSVLineSplit)

	buf := make([]byte, 0, initBufSizeBytes)
	scanner.Buffer(buf, maxBufSizeBytes)

	for scanner.Scan() {
		line := scanner.Text() + "\n"
		numBytes := len(line)

		lineStraddlesSplit := currFileBytes+numBytes > maxBytesPerFile
		if lineStraddlesSplit {
			// Time for a new file.
			f, err := genNextFile()
			if err != nil {
				return fmt.Errorf("getting next file: %v", err)
			}
			w = bufio.NewWriter(f)
			currFileBytes = 0
		}

		_, err := w.WriteString(line)
		w.Flush()
		if err != nil {
			return fmt.Errorf("writing line: %v", err)
		}
		currFileBytes += numBytes
	}
	err := scanner.Err()
	if err != nil {
		return fmt.Errorf("scanning: %v", err)
	}

	return nil
}

var numLineBytes = flag.Int("line-bytes", -1, "put at most SIZE bytes of records per output file")
var suffixLength = flag.Int("suffix-length", 2, "generate suffixes of length N")
var numericSuffixes = flag.Int("numeric-suffixes", 0, "use numeric suffixes starting at X")
var additionalSuffix = flag.String("additional-suffix", "", "append an additional SUFFIX to file names")
var prefix = flag.String("prefix", "", "prefix for file names")
var verbose = flag.Bool("verbose", false, "generate verbose output")

// NextFileName generates the file name for the split file in position fileNum.
func NextFileName(fileNum int) (string, error) {
	num := fmt.Sprintf("%d", fileNum)
	numPaddingChars := *suffixLength - len(num)
	if numPaddingChars < 0 {
		return "", fmt.Errorf("file number longer than suffix size (%d): %s", *suffixLength, num)
	}
	paddedNum := strings.Repeat("0", numPaddingChars) + num

	return fmt.Sprintf("%s%s%s", *prefix, paddedNum, *additionalSuffix), nil
}

func main() {
	flag.Parse()

	if *numLineBytes < 1 {
		fmt.Fprintf(os.Stderr, "must set -line-bytes to a positive integer (%d)\n", *numLineBytes)
		os.Exit(1)
	}

	fileNum := *numericSuffixes

	var activeFile *os.File
	genNext := func() (io.Writer, error) {
		if activeFile != nil {
			err := activeFile.Close()
			if err != nil {
				return nil, fmt.Errorf("closing previous active file: %v", err)
			}
		}

		name, err := NextFileName(fileNum)
		if err != nil {
			return nil, err
		}

		activeFile, err := os.Create(name)
		if err != nil {
			return nil, fmt.Errorf("creating new file %s: %v", name, err)
		}

		fileNum += 1

		if *verbose {
			fmt.Fprintf(os.Stderr, "opened new file for writing: %s\n", name)
		}

		return activeFile, nil
	}

	err := Split(os.Stdin, *numLineBytes, genNext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "splitting input: %v\n", err)
		os.Exit(1)
	}

	if activeFile != nil {
		err = activeFile.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "closing file: %v\n", err)
			os.Exit(1)
		}
	}
}
