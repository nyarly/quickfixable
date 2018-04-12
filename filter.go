package main

import (
	"bufio"
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
	projRE             *regexp.Regexp
}

func newFilter(gopath, projPath, packPath string, in io.Reader) *filter {
	proj := filepath.Join(gopath, "src", projPath)
	pack := filepath.Join(proj, packPath)

	var projRE = regexp.MustCompile(`^\s*(` + string(proj) + `\S*).*`)

	return &filter{
		buf:    bufio.NewReader(in),
		gopath: gopath,
		proj:   proj,
		projRE: projRE,
		pack:   pack,
		extra:  []byte(fmt.Sprintf("BEGIN   %s\n", pack)),
	}
}

var failHeader = regexp.MustCompile(`^--- FAIL: `)
var failExit = regexp.MustCompile("^FAIL\t")
var panicHeader = regexp.MustCompile(`^panic: `)
var carriageReturn = regexp.MustCompile(".*\r\\s*")
var goroutineRE = regexp.MustCompile(`^goroutine\s+\d+`)

func (f *filter) Read(p []byte) (n int, err error) {
	var part []byte
	for len(f.extra) < cap(p) {
		var err error
		part, err = f.buf.ReadBytes('\n')
		if err != nil {
			return 0, err
		}

		part := f.handleLine(part)
		spew.Dump(string(part), f.infail, f.inpanic)

		f.extra = append(f.extra, part...)

	}

	return f.flush(p)
}

func (f *filter) handleLine(line []byte) []byte {
	spew.Dump(string(line))
	switch {
	default:
		return nil
	case failExit.Match(line):
		f.infail = false
		f.inpanic = false
		return nil
	case panicHeader.Match(line):
		f.inpanic = true
		return debug("ph", line)
	case failHeader.Match(line):
		f.infail = true
		return debug("fh", line)
	case f.inpanic && goroutineRE.Match(line):
		return debug("ip", line)
	case f.inpanic:
		if m := f.projRE.FindSubmatch(line); m != nil {
			return debug("ipp", append(m[1], '\n'))
		} else {
			return nil
		}
	case f.infail:
		return debug("if", line)
	}
}

func debug(prefix string, line []byte) []byte {
	return append([]byte(prefix+": "), line...)
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
	n = copy(p, f.extra)
	f.extra = f.extra[n:]
	return
}
