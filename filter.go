package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"

	"github.com/davecgh/go-spew/spew"
)

type filter struct {
	extra              []byte
	buf                *bufio.Reader
	gopath, proj, pack string
	infail, inpanic    bool
	goroutineRE,
	projRE *regexp.Regexp
}

func newFilter(gopath, projPath, packPath string, in io.Reader) *filter {
	proj := filepath.Join(gopath, "src", projPath)
	pack := filepath.Join(proj, packPath)

	return &filter{
		buf:    bufio.NewReader(in),
		gopath: gopath,
		proj:   proj,
		pack:   pack,
		extra:  []byte(fmt.Sprintf("BEGIN   %s\n", pack)),
	}
}

var failHeader = regexp.MustCompile(`^--- FAIL: `)
var failExit = regexp.MustCompile("^FAIL\t")
var panicHeader = regexp.MustCompile(`^panic: `)
var carriageReturn = regexp.MustCompile(".*\r\\s*")
var goroutineRE = regexp.MustCompile(`^goroutine\s+\d+`)
var projRE = regexp.MustCompile(`^\s*(` + string(proj) + `\S*).*`)

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
		//log.Print("read ", string(part))

		if bytes.Index(part, failExit) == 0 {
			f.infail = false
			f.inpanic = false
			break
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
			return f.flush(p)
		}

		if bytes.Index(part, failHeader) == 0 {
			f.infail = true
		}

		if f.inpanic {
			if goroutineRE.Match(part) {
				break
			}
			if m := projRE.FindSubmatch(part); m != nil {
				part = append(m[1], '\n')
				break
			}
			continue
		}

		if f.infail {
			f.extra = append(f.extra, part...)
			if err := f.filterLine(); err != nil {
				return 0, err
			}
			return f.flush(p)
		}
	}

	f.extra = append(f.extra, part...)
	return f.flush(p)
}

func (f *filter) handleLine(line []byte) (keep bool, line []byte) {
	switch {
	default:
		keep = true
	case failExit.Match(line):
		f.infail = false
		f.inpanic = false
		keep = false
	case panicHeader.Match(line):
		f.inpanic = true
		keep = true
	}

	return
}

func (f *filter) grabLine() error {
	part, err := f.buf.ReadBytes('\n')
	if err != nil {
		return err
	}
	//log.Print("grab ", string(part))
	f.extra = append(f.extra, part...)
	return nil
}

func (f *filter) filterLine() error {
	part, err := f.buf.ReadBytes('\n')
	if err != nil {
		return err
	}
	part = carriageReturn.ReplaceAll(part, []byte{})

	//log.Print("grab ", string(part))
	f.extra = append(f.extra, part...)
	return nil
}

func (f *filter) flush(p []byte) (n int, err error) {
	spew.Dump(string(f.extra))
	n = copy(p, f.extra)
	f.extra = f.extra[n:]
	return
}
