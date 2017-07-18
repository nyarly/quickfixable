package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type filter struct {
	extra              []byte
	buf                *bufio.Reader
	gopath, proj, pack string
	infail, inpanic    bool
	goroutineRE,
	projRE *regexp.Regexp
}

var failHeader = []byte("--- FAIL: ")
var failExit = []byte("FAIL\t")
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

	var defWorkingGopath, defProjectPath string
	for _, gop := range strings.Split(path, ":") {
		rel, err := filepath.Rel(gop, pwd)
		if err == nil {
			if strings.Index(rel, ".") != 0 {
				defWorkingGopath = gop
				defProjectPath = rel
				break
			}
		}
	}

	var workingGopath, projectPath, packagePath string
	flag.StringVar(&workingGopath, "gopath", defWorkingGopath, "The Go path.")
	flag.StringVar(&projectPath, "projectpath", defProjectPath, "The path to the project under the Go path.")
	flag.StringVar(&packagePath, "packagePath", "", "The path to the tested package under the project.")

	flag.Parse()

	if workingGopath == "" {
		log.Fatalf("Could not determine a current working Go path in %q", path)
	}

	filter := newFilter(workingGopath, projectPath, packagePath, os.Stdin)
	_, err = io.Copy(os.Stdout, filter)
	if err != nil {
		log.Fatal(err)
	}
}

func newFilter(gopath, projPath, packPath string, in io.Reader) *filter {
	proj := filepath.Join(gopath, "src", projPath)
	pack := filepath.Join(proj, packPath)

	return &filter{
		buf:         bufio.NewReader(in),
		gopath:      gopath,
		proj:        proj,
		pack:        pack,
		goroutineRE: regexp.MustCompile(`^goroutine\s+\d+`),
		projRE:      regexp.MustCompile(`^\s*(` + string(proj) + `\S*).*`),
		extra:       []byte(fmt.Sprintf("BEGIN   %s\n", pack)),
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
			if f.goroutineRE.Match(part) {
				break
			}
			if m := f.projRE.FindSubmatch(part); m != nil {
				part = append(m[1], '\n')
				break
			}
			continue
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
	//log.Print("grab ", string(part))
	f.extra = append(f.extra, part...)
	return nil
}

func (f *filter) flush(p []byte) (n int, err error) {
	//log.Print("flush ", string(f.extra))
	n = copy(p, f.extra)
	f.extra = f.extra[n:]
	return
}
