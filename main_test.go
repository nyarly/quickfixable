package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFilter(t *testing.T) {
	cases, err := filepath.Glob("testdata/*")
	if err != nil {
		t.Fatal(err)
	}

	testcase := func(dir string) {
		name := filepath.Base(dir)
		t.Run(name, func(t *testing.T) {
			inpath := filepath.Join(dir, "in.txt")
			outpath := filepath.Join(dir, "out.txt")

			in, err := os.Open(inpath)
			if err != nil {
				t.Fatal(inpath, err)
			}

			expected, err := ioutil.ReadFile(outpath)
			if err != nil {
				t.Fatal(outpath, err)
			}

			out := &bytes.Buffer{}

			err = runFilter("/home/judson/golang", in, out)
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

				t.Errorf("Filter output didn't match %q. Results recorded in %q.", outpath, errfile.Name())
			}
		})
	}

	for _, dir := range cases {
		testcase(dir)
	}
}
