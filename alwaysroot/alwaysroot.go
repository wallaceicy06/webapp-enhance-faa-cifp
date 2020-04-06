// Package alwaysroot will ensure that the filepaths used will be relative
// to the project root.
//
// Ripped off from https://brandur.org/fragments/testing-go-project-root.
// Thank you very much! The fact that Go does not handle this is just silly.
package alwaysroot

import (
	"log"
	"os"
	"path"
	"runtime"
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		log.Printf("Could not change directory, some tests may fail: %v", err)
	}
}
