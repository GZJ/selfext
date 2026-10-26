// stub is compiled once per target at release time, then reused for every archive.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gzj/selfext/internal/bundle"
	"github.com/gzj/selfext/internal/extract"
)

func run() error {
	destination := flag.String("directory", "", "destination directory (default: archive name in current directory)")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flag.Args())
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	f, err := os.Open(exe)
	if err != nil {
		return err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	payload, err := bundle.Read(f, stat.Size())
	if err != nil {
		return err
	}
	if *destination == "" {
		*destination = payload.Metadata.Directory
	}
	if err := extract.Reader(payload.Archive, payload.Metadata.ArchiveName, *destination); err != nil {
		return err
	}
	fmt.Printf("Extracted to %s\n", *destination)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "selfext:", err)
		os.Exit(1)
	}
}
