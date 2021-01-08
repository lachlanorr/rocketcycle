// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var header = `// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
`

func fileHasHeaderIssue(path string) bool {
	ext := filepath.Ext(path)
	if ext == ".go" || ext == ".proto" {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("Failed to read %s, err: %s\n", path, err.Error())
			return true
		}
		content := string(bytes)

		if !strings.Contains(content, header) {
			fmt.Printf("Bad header: %s\n", path)
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
