package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	path, got := os.LookupEnv("GOPATH")
	if !got {
		log.Fatal("GOPATH unset!")
	}
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %v", err)
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
