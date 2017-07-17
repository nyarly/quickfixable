package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type filter struct {
	extra           []byte
	buf             *bufio.Reader
	path            string
	infail, inpanic bool
}

var failHeader = []byte("--- FAIL: ")
var failExit = []byte("FAIL    ")
var panicHeader = []byte("panic: ")

func main() {
	path, got := os.LookupEnv("GOPATH")
	if !got {
		log.Fatal("GOPATH unset!")
	}
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error geting current working directory: %v", err)
	}

	var workingGopath, projectPath string
	for _, gop := range strings.Split(path, ":") {
		rel, err := filepath.Rel(gop, pwd)
		if err == nil {
			if strings.Index(rel, ".") != 0 {
				workingGopath = gop
				projectPath = rel
				break
			}
		}
	}

	if workingGopath == "" {
		log.Fatalf("Could not determine a current working Go path in %q", path)
	}

	err = runFilter(workingGopath, os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func runFilter(workingGopath string, in io.Reader, out io.Writer) error {
	_, err := io.Copy(out, newFilter(workingGopath, in))
	return err
}

func newFilter(path string, in io.Reader) *filter {
	return &filter{
		buf:   bufio.NewReader(in),
		path:  path,
		extra: []byte(fmt.Sprintf("BEGIN %s\n", path)),
	}
}

func (f *filter) Read(p []byte) (n int, err error) {

	if len(f.extra) > 0 {
		return f.flush(p)
	}

	var part []byte
	for {
		var err error
		part, err = f.buf.ReadBytes('\n')
		if err != nil {
			return 0, err
		}

		if bytes.Index(part, panicHeader) == 0 {
			f.extra = append(f.extra, part...)
			// skip extra panic notification
			_, err = f.buf.ReadBytes('\n')
			if err != nil {
				return 0, err
			}
			// grab panic cause
			if err := f.grabLine(); err != nil {
				return 0, err
			}
			// grab spacer line
			if err := f.grabLine(); err != nil {
				return 0, err
			}
			f.inpanic = true
			break
		}

		if bytes.Index(part, failHeader) == 0 {
			f.infail = true
		}

		log.Print(f.infail, string(part))

		if bytes.Index(part, failExit) == 0 {
			f.infail = false
			f.inpanic = false
			break
		}

		if f.infail {
			break
		}
	}

	f.extra = append(f.extra, part...)
	return f.flush(p)
}

func (f *filter) grabLine() error {
	part, err := f.buf.ReadBytes('\n')
	if err != nil {
		return err
	}
	f.extra = append(f.extra, part...)
	return nil
}

func (f *filter) flush(p []byte) (n int, err error) {
	n = copy(p, f.extra)
	f.extra = f.extra[n:]
	return
}
