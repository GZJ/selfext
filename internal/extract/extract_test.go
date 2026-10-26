package extract

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func archive(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.zip")
	f, e := os.Create(p)
	if e != nil {
		t.Fatal(e)
	}
	w := zip.NewWriter(f)
	entry, e := w.Create(name)
	if e != nil {
		t.Fatal(e)
	}
	entry.Write([]byte("hello"))
	w.Close()
	f.Close()
	return p
}
func TestRegularFile(t *testing.T) {
	dst := t.TempDir()
	if e := Unarchive(archive(t, "a/b.txt"), dst); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(dst, "a/b.txt"))
	if e != nil || string(b) != "hello" {
		t.Fatal(string(b), e)
	}
}
func TestUnsafePaths(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "a/../../escape", "C:/escape", "a\\b"} {
		t.Run(name, func(t *testing.T) {
			if e := Unarchive(archive(t, name), t.TempDir()); e == nil {
				t.Fatal("accepted unsafe name")
			}
		})
	}
}
func TestExistingSymlink(t *testing.T) {
	dst := t.TempDir()
	outside := t.TempDir()
	if e := os.Symlink(outside, filepath.Join(dst, "a")); e != nil {
		t.Skip(e)
	}
	if e := Unarchive(archive(t, "a/escape"), dst); e == nil {
		t.Fatal("followed outside link")
	}
	if _, e := os.Stat(filepath.Join(outside, "escape")); !os.IsNotExist(e) {
		t.Fatal("outside file written")
	}
}
func TestExistingHardlink(t *testing.T) {
	dst := t.TempDir()
	outside := filepath.Join(t.TempDir(), "original")
	os.WriteFile(outside, []byte("original"), 0600)
	if e := os.Link(outside, filepath.Join(dst, "file")); e != nil {
		t.Skip(e)
	}
	if e := Unarchive(archive(t, "file"), dst); e == nil {
		t.Fatal("overwrote hardlink")
	}
	b, _ := os.ReadFile(outside)
	if string(b) != "original" {
		t.Fatal("outside file changed")
	}
}

func TestArchiveSymlink(t *testing.T) {
	p := filepath.Join(t.TempDir(), "link.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	h := &zip.FileHeader{Name: "link"}
	h.SetMode(os.ModeSymlink | 0777)
	w, err := z.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("../outside"))
	z.Close()
	f.Close()
	if err := Unarchive(p, t.TempDir()); err == nil {
		t.Fatal("accepted symlink entry")
	}
}

func TestExistingFile(t *testing.T) {
	dst := t.TempDir()
	p := filepath.Join(dst, "file")
	if err := os.WriteFile(p, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Unarchive(archive(t, "file"), dst); err == nil {
		t.Fatal("overwrote existing file")
	}
	got, err := os.ReadFile(p)
	if err != nil || string(got) != "original" {
		t.Fatalf("%q: %v", got, err)
	}
}
