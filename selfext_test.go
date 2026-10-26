package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIUsage(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version"}} {
		if err := run(args, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{nil, {"a", "b"}, {"--archive", "a", "b"}, {"--unknown"}} {
		if err := run(args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestRejectNonArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{path}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("accepted text file")
	}
}

func zipFile(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	w, err := z.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "test.zip")
	if err := os.WriteFile(p, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUnsupportedTarget(t *testing.T) {
	if err := run([]string{zipFile(t), "--os", "darwin", "--arch", "386"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("accepted unsupported target")
	}
}
