package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

func TestFilter(t *testing.T) {
	/*
		cases, err := filepath.Glob("testdata/*")
		if err != nil {
			t.Fatal(err)
		}
	*/

	escapes := regexp.MustCompile(`\r`)

	escape := func(s string) string {
		return escapes.ReplaceAllStringFunc(s, func(r string) string {
			return "^M"
		})
	}

	testcase := func(gp, wp, pp, dir string) {
		name := dir
		t.Run(name, func(t *testing.T) {
			inpath := filepath.Join("testdata", dir, "in.txt")
			outpath := filepath.Join("testdata", dir, "out.txt")

			in, err := os.Open(inpath)
			if err != nil {
				t.Fatal(inpath, err)
			}

			expected, err := ioutil.ReadFile(outpath)
			if err != nil {
				t.Fatal(outpath, err)
			}

			out := &bytes.Buffer{}

			filter := newFilter(gp, wp, pp, in)
			_, err = io.Copy(out, filter)

			if err != nil {
				t.Fatal(err)
			}

			actual := out.Bytes()

			if !bytes.Equal(expected, actual) {
				errfile, err := ioutil.TempFile("", name)
				if err != nil {
					t.Fatal("opening tempfile for output", err)
				}
				io.Copy(errfile, bytes.NewReader(actual))

				ud := difflib.UnifiedDiff{
					A:        difflib.SplitLines(escape(string(expected))),
					B:        difflib.SplitLines(escape(string(actual))),
					FromFile: inpath,
					FromDate: "",
					ToFile:   outpath,
					ToDate:   "",
					Context:  1,
				}
				diff, _ := difflib.GetUnifiedDiffString(ud)

				t.Errorf("Filter output didn't match %q. Results recorded in %q.\nDiff: %s", outpath, errfile.Name(), diff)
			}
		})
	}

	testcase("/home/judson/golang", "github.com/opentable/sous", "ext/singularity", "one")
	testcase("/home/judson/golang", "github.com/opentable/sous", "ext/singularity", "two")
}
