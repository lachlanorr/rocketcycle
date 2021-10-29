// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Prepend the firs few chars so we don't match the license in this file and therefore
// exclude check_file_headers.go from the checking.
var header = "// " + `This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
`

var incl *regexp.Regexp = regexp.MustCompile(`\.(go|proto)$`)
var excl *regexp.Regexp = regexp.MustCompile(`(\.pb\.gw\.go|_grpc\.pb\.go|protobuf_internal_str_strings\.go)$`)

func fileHasHeaderIssue(path string) bool {
	if incl.MatchString(path) && !excl.MatchString(path) {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("Failed to read %s, err: %s\n", path, err.Error())
			return true
		}
		content := string(bytes)
		if !strings.Contains(content, header) {
			fmt.Printf("%s:1:1: bad license header\n", path)
			return true
		}
	}
	return false
}

func dirHasHeaderIssue(root string) bool {
	foundError := false

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if !info.IsDir() {
				foundError = fileHasHeaderIssue(path) || foundError
			}
		}
		return nil
	})

	return foundError
}

func main() {
	roots := os.Args[1:]
	if len(roots) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		roots = append(roots, wd)
	}

	foundError := false
	for _, root := range roots {
		foundError = dirHasHeaderIssue(root) || foundError
	}

	if foundError {
		os.Exit(1)
	}
}
