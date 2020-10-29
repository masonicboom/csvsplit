package main

import "fmt"
import "io"
import "encoding/csv"
import "strings"
import "flag"
import "bufio"
import "os"

func Split(in io.Reader, maxBytesPerFile int, genNextFile func() (io.Writer, error)) error {
	var w *bufio.Writer
	currFileBytes := maxBytesPerFile // This forces a new file to be generated initially.

	r := csv.NewReader(in)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		line := strings.Join(rec, ",") + "\n"
		numBytes := len(line) // TODO: confirm this counts bytes, not multi-byte runes.

		if currFileBytes+numBytes > maxBytesPerFile {
			// Time for a new file.
			f, err := genNextFile()
			if err != nil {
				return fmt.Errorf("getting next file: %v", err)
			}
			w = bufio.NewWriter(f)
			currFileBytes = 0
		}

		_, err = w.WriteString(line)
		w.Flush()
		if err != nil {
			return fmt.Errorf("writing line: %v", err)
		}
		currFileBytes += numBytes
	}

	return nil
}

/* example split command I'm using
split \
  --line-bytes=200000000 \
  --numeric-suffixes=0 \
  --additional-suffix=.csv \
  --suffix-length=4 \
  --verbose \
  "${STAGING_DIR}/${TABLE}.csv" "${STAGING_DIR}/${TABLE}_chunk"
*/

var numLineBytes = flag.Int("-line-bytes", -1, "put at most SIZE bytes of records per output file")
var suffixLength = flag.Int("-suffix-length", 2, "generate suffixes of length N")
var numericSuffixes = flag.Int("-numeric-suffixes", 0, "use numeric suffixes starting at X")
var additionalSuffix = flag.String("-additional-suffix", "", "append an additional SUFFIX to file names")

func NextFileName(fileNum int) (string, error) {
	num := fmt.Sprintf("%d", fileNum)
	numPaddingChars := *suffixLength - len(num)
	if numPaddingChars < 0 {
		return "", fmt.Errorf("file number longer than suffix size (%d): %s", *suffixLength, num)
	}
	paddedNum := num + strings.Repeat("0", numPaddingChars)

	return fmt.Sprintf("%s%s", paddedNum, *additionalSuffix), nil
}

func main() {
	flag.Parse()

	if *numLineBytes < 1 {
		fmt.Fprintf(os.Stderr, "must set --line-bytes to a positive integer (%d)\n", *numLineBytes)
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

		f, err := os.Create(name)
		if err != nil {
			return nil, fmt.Errorf("creating new file %s: %v", name, err)
		}

		fileNum += 1

		return f, nil
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

	fmt.Println(*numLineBytes)
}

