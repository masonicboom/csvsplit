package main

import "testing"
import "strings"
import "bytes"
import "io"
import "bufio"

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
	// Variables:
	// single vs. multi line
	// single vs. multi column
	// quoted vs. unquoted
	// first line vs. last line vs. middle vs. boundary
	// greater than file size limit vs. less than

	cases := []struct {
		input           string
		maxBytesPerFile int
		expected        []string
	}{
		{
			"a,b,c",
			100,
			[]string{
				"a,b,c\n",
			},
		},
		{
			"a,b,c\nd,e,f",
			100,
			[]string{
				"a,b,c\nd,e,f\n",
			},
		},
		{
			"a,b,c\nd,e,f",
			6,
			[]string{
				"a,b,c\n",
				"d,e,f\n",
			},
		},
		{
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
			"a,b\nc,d",
			[]string{
				"a,b",
				"c,d",
			},
		},
		{
			"a,b\nc,\"d\ne\"",
			[]string{
				"a,b",
				"c,\"d\ne\"",
			},
		},
		{
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
