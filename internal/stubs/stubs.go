// Package stubs holds the precompiled extractors shipped with selfext.
package stubs

import (
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"slices"

	"github.com/gzj/selfext/internal/targets"
)

// Targets is the release platform matrix, shared by the build tool and CLI.
var Targets = targets.All

//go:embed data/*
var files embed.FS

func Open(goos, goarch string) (io.ReadCloser, error) {
	target := goos + "/" + goarch
	if !slices.Contains(Targets, target) {
		return nil, fmt.Errorf("unsupported target %s", target)
	}
	f, err := files.Open("data/" + goos + "-" + goarch + ".gz")
	if err != nil {
		return nil, fmt.Errorf("extractor for %s is missing from this build; reinstall an official release (maintainers: run `make stubs`)", target)
	}
	z, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &reader{Reader: z, file: f}, nil
}

type reader struct {
	*gzip.Reader
	file io.Closer
}

func (r *reader) Close() error {
	err := r.Reader.Close()
	if e := r.file.Close(); err == nil {
		err = e
	}
	return err
}
