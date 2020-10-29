package main

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
)

func SplitToBuffers(in io.Reader, maxBytesPerFile int) ([]*bytes.Buffer, error) {
	files := []*bytes.Buffer{}

	genNext := func() (io.Writer, error) {
		b := &bytes.Buffer{}
		files = append(files, b)
		return b, nil
	}

	err := Split(in, maxBytesPerFile, genNext)
	return files, err
}

func TestSplit(t *testing.T) {
	cases := []struct {
		input           string
		maxBytesPerFile int
		expected        []string
	}{
		{
			// Single row.
			"a,b,c",
			100,
			[]string{
				"a,b,c\n",
			},
		},
		{
			// Two rows, under split limit.
			"a,b,c\nd,e,f",
			100,
			[]string{
				"a,b,c\nd,e,f\n",
			},
		},
		{
			// Two rows, above split limit.
			"a,b,c\nd,e,f",
			6,
			[]string{
				"a,b,c\n",
				"d,e,f\n",
			},
		},
		{
			// 3 rows, above split limit.
			"a,b,c\nd,e,f\ng,h,i",
			6,
			[]string{
				"a,b,c\n",
				"d,e,f\n",
				"g,h,i\n",
			},
		},
		{
			// Quoted embedded newline, above split limit.
			"a,b,\"c\nasdfasdf\"\nd,e,f",
			6,
			[]string{
				"a,b,\"c\nasdfasdf\"\n",
				"d,e,f\n",
			},
		},
	}

	for _, tc := range cases {
		inr := strings.NewReader(tc.input)
		out, err := SplitToBuffers(inr, tc.maxBytesPerFile)

		if err != nil {
			t.Errorf("failed on input \"%s\": %v", tc.input, err)
			goto nextcase
		}

		if len(out) != len(tc.expected) {
			t.Errorf("mismatching number of files: %d vs. %d expected", len(out), len(tc.expected))
			goto nextcase
		}

		for i := range out {
			fa, fe := out[i].String(), tc.expected[i]
			if fa != fe {
				t.Errorf("mismatching files: %s vs. %s expected", fa, fe)
				goto nextcase
			}
		}
	nextcase:
	}
}

func TestQuotedCSVLineSplit(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			// Newline separating rows.
			"a,b\nc,d",
			[]string{
				"a,b",
				"c,d",
			},
		},
		{
			// Newline embedded within quoted field (not a row separator).
			"c,\"d\ne\"",
			[]string{
				"c,\"d\ne\"",
			},
		},
		{
			// Escaped quote within quoted field.
			"a,b,\"c\"\"\"",
			[]string{
				"a,b,\"c\"\"\"",
			},
		},
	}

	for _, tc := range cases {
		scanner := bufio.NewScanner(strings.NewReader(tc.input))
		scanner.Split(QuotedCSVLineSplit)

		i := 0
		for scanner.Scan() {
			line := scanner.Text()
			expc := tc.expected[i]
			if line != expc {
				t.Errorf("mismatching lines: %s vs. %s expected", line, expc)
			}
			i += 1
		}
	}
}
