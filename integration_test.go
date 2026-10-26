//go:build integration

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gzj/selfext/internal/bundle"
	"github.com/gzj/selfext/internal/stubs"
)

func integrationArchive(t *testing.T, dir, name, entry string, tarGz bool) string {
	t.Helper()
	var buf bytes.Buffer
	if tarGz {
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(&tar.Header{Name: entry, Mode: 0644, Size: 5}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		z := zip.NewWriter(&buf)
		w, err := z.Create(entry)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestIntegration(t *testing.T) {
	work := t.TempDir()
	packager := filepath.Join(work, "selfext")
	if runtime.GOOS == "windows" {
		packager += ".exe"
	}
	build := exec.Command("go", "build", "-o", packager, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build packager: %s: %v", out, err)
	}
	// No external tool (including Go) can be resolved by the distributed programs.
	cleanEnv := []string{}
	for _, value := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(value, "=", 2)[0])
		if key != "PATH" && key != "XDG_CONFIG_HOME" {
			cleanEnv = append(cleanEnv, value)
		}
	}
	emptyPath := filepath.Join(work, "empty-path")
	if err := os.Mkdir(emptyPath, 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(work, "config")
	cleanEnv = append(cleanEnv, "PATH="+emptyPath, "XDG_CONFIG_HOME="+config)
	runBinary := func(dir, bin string, args ...string) ([]byte, error) {
		cmd := exec.Command(bin, args...)
		cmd.Dir, cmd.Env = dir, cleanEnv
		return cmd.CombinedOutput()
	}
	for _, arg := range []string{"--version", "--help"} {
		if out, err := runBinary(work, packager, arg); err != nil {
			t.Fatalf("%s: %s: %v", arg, out, err)
		}
	}
	for _, compressed := range []bool{false, true} {
		t.Run(fmt.Sprintf("native-targz-%v", compressed), func(t *testing.T) {
			dir := t.TempDir()
			name := "资料 & quote' archive.zip"
			if compressed {
				name = "资料 & quote' archive.tar.gz"
			}
			archive := integrationArchive(t, dir, name, "nested/file.txt", compressed)
			if out, err := runBinary(dir, packager, archive); err != nil {
				t.Fatalf("package: %s: %v", out, err)
			}
			exe := archive + ".run"
			if runtime.GOOS == "windows" {
				exe = archive + ".exe"
			}
			if out, err := runBinary(dir, exe); err != nil {
				t.Fatalf("extract: %s: %v", out, err)
			}
			actual, err := os.ReadFile(filepath.Join(dir, bundle.Directory(name), "nested", "file.txt"))
			if err != nil || string(actual) != "hello" {
				t.Fatalf("%q: %v", actual, err)
			}
			if out, err := runBinary(dir, exe); err == nil {
				t.Fatalf("overwrote extracted file: %s", out)
			}
			custom := filepath.Join(dir, "custom")
			if out, err := runBinary(dir, exe, "--directory", custom); err != nil {
				t.Fatalf("custom directory: %s: %v", out, err)
			}
			before, err := os.ReadFile(archive)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := runBinary(dir, packager, archive, "-o", archive); err == nil {
				t.Fatal("overwrote input")
			}
			after, err := os.ReadFile(archive)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("input changed")
			}
			original, err := os.ReadFile(exe)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := runBinary(dir, packager, archive); err == nil {
				t.Fatal("overwrote output")
			}
			same, err := os.ReadFile(exe)
			if err != nil || !bytes.Equal(original, same) {
				t.Fatal("output changed")
			}
		})
	}
	archive := integrationArchive(t, work, "platforms.zip", "file.txt", false)
	for _, target := range stubs.Targets {
		t.Run(target, func(t *testing.T) {
			parts := strings.Split(target, "/")
			output := filepath.Join(t.TempDir(), "output")
			if out, err := runBinary(work, packager, archive, "--os", parts[0], "--arch", parts[1], "-o", output); err != nil {
				t.Fatalf("package: %s: %v", out, err)
			}
			f, err := os.Open(output)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			info, err := f.Stat()
			if err != nil {
				t.Fatal(err)
			}
			payload, err := bundle.Read(f, info.Size())
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(payload.Archive)
			original, readErr := os.ReadFile(archive)
			if err != nil || readErr != nil || !bytes.Equal(data, original) {
				t.Fatal("archive not preserved")
			}
			switch parts[0] {
			case "linux":
				e, err := elf.NewFile(f)
				if err != nil {
					t.Fatal(err)
				}
				want := map[string]elf.Machine{"386": elf.EM_386, "amd64": elf.EM_X86_64, "arm64": elf.EM_AARCH64}[parts[1]]
				if e.Machine != want {
					t.Fatal(e.Machine)
				}
			case "windows":
				p, err := pe.NewFile(f)
				if err != nil {
					t.Fatal(err)
				}
				want := map[string]uint16{"386": pe.IMAGE_FILE_MACHINE_I386, "amd64": pe.IMAGE_FILE_MACHINE_AMD64, "arm64": pe.IMAGE_FILE_MACHINE_ARM64}[parts[1]]
				if p.Machine != want {
					t.Fatal(p.Machine)
				}
			case "darwin":
				m, err := macho.NewFile(f)
				if err != nil {
					t.Fatal(err)
				}
				want := map[string]macho.Cpu{"amd64": macho.CpuAmd64, "arm64": macho.CpuArm64}[parts[1]]
				if m.Cpu != want {
					t.Fatal(m.Cpu)
				}
				if parts[1] == "arm64" {
					verifyMachSignature(t, f, m, info.Size())
					if runtime.GOOS == "darwin" {
						check := exec.Command("/usr/bin/codesign", "--verify", "--strict", output)
						if out, err := check.CombinedOutput(); err != nil {
							t.Fatalf("codesign verification: %s: %v", out, err)
						}
					}
				}
			}
		})
	}
	t.Run("unsafe-entry", func(t *testing.T) {
		dir := t.TempDir()
		archive := integrationArchive(t, dir, "unsafe.zip", "../escape", false)
		exe := filepath.Join(dir, "unsafe.run")
		if runtime.GOOS == "windows" {
			exe += ".exe"
		}
		if out, err := runBinary(dir, packager, archive, "-o", exe); err != nil {
			t.Fatalf("%s: %v", out, err)
		}
		if out, err := runBinary(dir, exe); err == nil {
			t.Fatalf("accepted unsafe entry: %s", out)
		}
		if _, err := os.Stat(filepath.Join(dir, "escape")); !os.IsNotExist(err) {
			t.Fatal("escaped destination")
		}
	})
	if runtime.GOOS == "linux" {
		if _, err := os.Stat(config); !os.IsNotExist(err) {
			t.Fatal("runtime created configuration/SDK directory")
		}
	}
}

// Parse the Apple signature independently of our signing implementation, checking
// every page hash and that the signed range includes the complete bundle.
func verifyMachSignature(t *testing.T, f *os.File, m *macho.File, size int64) {
	t.Helper()
	var off, length uint32
	for _, load := range m.Loads {
		raw := load.Raw()
		if m.ByteOrder.Uint32(raw) == 0x1d {
			off = m.ByteOrder.Uint32(raw[8:12])
			length = m.ByteOrder.Uint32(raw[12:16])
		}
	}
	if off == 0 || int64(off)+int64(length) != size {
		t.Fatal("signature bounds")
	}
	link := m.Segment("__LINKEDIT")
	if link == nil || link.Offset+link.Filesz != uint64(size) {
		t.Fatal("linkedit bounds")
	}
	sig := make([]byte, length)
	if _, err := f.ReadAt(sig, int64(off)); err != nil {
		t.Fatal(err)
	}
	be := binary.BigEndian
	if be.Uint32(sig[:4]) != 0xfade0cc0 || be.Uint32(sig[4:8]) != length || be.Uint32(sig[8:12]) != 1 {
		t.Fatal("superblob")
	}
	cd := sig[be.Uint32(sig[16:20]):]
	if be.Uint32(cd[:4]) != 0xfade0c02 || be.Uint32(cd[32:36]) != off || cd[36] != 32 || cd[37] != 2 || cd[39] != 12 {
		t.Fatal("code directory")
	}
	hashOffset, slots := be.Uint32(cd[16:20]), be.Uint32(cd[28:32])
	if slots != (off+4095)/4096 {
		t.Fatal("page count")
	}
	for page := uint32(0); page < slots; page++ {
		n := min(uint32(4096), off-page*4096)
		data := make([]byte, n)
		if _, err := f.ReadAt(data, int64(page)*4096); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if !bytes.Equal(sum[:], cd[hashOffset+page*32:hashOffset+(page+1)*32]) {
			t.Fatalf("signature page %d mismatch", page)
		}
	}
}
