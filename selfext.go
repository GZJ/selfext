package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gzj/selfext/internal/bundle"
	"github.com/gzj/selfext/internal/stubs"
	"github.com/gzj/selfext/internal/version"
	"github.com/mholt/archives"
	flag "github.com/spf13/pflag"
)

func expandPath(p string) (string, error) {
	if p == "" {
		return "", errors.New("path cannot be empty")
	}
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			p = home
		} else {
			p = filepath.Join(home, p[2:])
		}
	}
	return filepath.Abs(p)
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("selfext", flag.ContinueOnError)
	flags.SetOutput(stderr)
	archive := flags.String("archive", "", "archive to package (e.g. .zip, .tar.gz)")
	targetOS := flags.String("os", runtime.GOOS, "target operating system")
	targetArch := flags.String("arch", runtime.GOARCH, "target architecture")
	output := flags.StringP("output", "o", "", "output executable path")
	showVersion := flags.BoolP("version", "V", false, "print version")
	verbose := flags.BoolP("verbose", "v", false, "show packaging details")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, version.Get().String())
		return nil
	}
	if flags.NArg() > 1 || (flags.NArg() == 1 && *archive != "") {
		return errors.New("provide one archive, either positional or --archive")
	}
	if flags.NArg() == 1 {
		*archive = flags.Arg(0)
	}
	if *archive == "" {
		return errors.New("archive required: selfext [--os linux --arch amd64] [-o output] archive.zip")
	}
	archivePath, err := expandPath(*archive)
	if err != nil {
		return err
	}
	source, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("archive must be a regular file")
	}
	format, _, err := archives.Identify(context.Background(), info.Name(), source)
	if err != nil {
		return fmt.Errorf("identify archive: %w", err)
	}
	if _, ok := format.(archives.Extractor); !ok {
		return errors.New("input is not a supported archive")
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return err
	}
	stub, err := stubs.Open(*targetOS, *targetArch)
	if err != nil {
		return err
	}
	defer stub.Close()
	if *output == "" {
		if *targetOS == "windows" {
			*output = archivePath + ".exe"
		} else {
			*output = archivePath + ".run"
		}
	}
	outputPath, err := expandPath(*output)
	if err != nil {
		return err
	}
	// Exclusive creation never truncates the input, existing output, or a link.
	out, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		out.Close()
		if !complete {
			os.Remove(outputPath)
		}
	}()
	if *verbose {
		fmt.Fprintf(stdout, "Target: %s/%s; archive: %d bytes; using embedded extractor\n", *targetOS, *targetArch, info.Size())
	}
	if err := bundle.Write(out, stub, source, bundle.Metadata{ArchiveName: info.Name(), Directory: bundle.Directory(info.Name())}); err != nil {
		return err
	}
	if err := out.Chmod(0755); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	complete = true
	fmt.Fprintf(stdout, "Created %s\n", outputPath)
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "selfext:", err)
		os.Exit(1)
	}
}
