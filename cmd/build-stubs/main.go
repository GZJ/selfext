// Build-time helper. Run from the repository root; never shipped to users.
package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gzj/selfext/internal/targets"
)

func build(target, dir string) error {
	parts := strings.Split(target, "/")
	bin := filepath.Join(dir, "stub")
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
	if runtime.GOOS == "windows" {
		goTool += ".exe"
	}
	cmd := exec.Command(goTool, "build", "-trimpath", "-ldflags=-s -w", "-o", bin, "./cmd/stub")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+parts[0], "GOARCH="+parts[1])
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	fmt.Println("Building extractor", target)
	if err := cmd.Run(); err != nil {
		return err
	}
	in, err := os.Open(bin)
	if err != nil {
		return err
	}
	defer in.Close()
	dst := filepath.Join("internal", "stubs", "data", strings.ReplaceAll(target, "/", "-")+".gz")
	out, err := os.CreateTemp(filepath.Dir(dst), "stub-*")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	defer out.Close()
	compressed, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := io.Copy(compressed, in); err != nil {
		return err
	}
	if err := compressed.Close(); err != nil {
		return err
	}
	if err := out.Chmod(0644); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(out.Name(), dst)
}

func run() error {
	dir, err := os.MkdirTemp("", "selfext-stubs-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	for _, target := range targets.All {
		if err := build(target, dir); err != nil {
			return err
		}
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
