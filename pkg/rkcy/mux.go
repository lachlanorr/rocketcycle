// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"net/http"
	"regexp"
)

type fileSystemPathable struct {
	fs   http.FileSystem
	re   *regexp.Regexp
	repl []byte
}

func newFileSystemPathable(fs http.FileSystem, pattern string, repl string) (*fileSystemPathable, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	fsP := fileSystemPathable{fs: fs, re: re, repl: []byte(repl)}
	return &fsP, nil
}

func (fsP *fileSystemPathable) Open(name string) (http.File, error) {
	newName := string(fsP.re.ReplaceAll([]byte(name), fsP.repl))
	return fsP.fs.Open(newName)
}

// Mux wrapper that supprts grpc Mux and http Mux for static files as
// well on same port
type serveMux struct {
	staticMux http.Handler
	apiMux    http.Handler
	fsP       *fileSystemPathable
}

func newServeMux(
	staticFs http.FileSystem,
	staticPattern string,
	staticReplace string,
	apiMux http.Handler,
) (*serveMux, error) {
	fsP, err := newFileSystemPathable(staticFs, staticPattern, staticReplace)
	if err != nil {
		return nil, err
	}
	staticMux := http.NewServeMux()
	staticMux.Handle("/", http.FileServer(fsP))

	mux := serveMux{staticMux, apiMux, fsP}
	return &mux, nil
}

func (mux *serveMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if mux.fsP.re.MatchString(r.URL.Path) {
		mux.staticMux.ServeHTTP(w, r)
	} else {
		mux.apiMux.ServeHTTP(w, r)
	}
}
