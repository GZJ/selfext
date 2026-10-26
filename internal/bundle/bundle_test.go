package bundle

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "bundle"))
	if err != nil {
		t.Fatal(err)
	}
	// Write needs a read/write file for Mach-O inspection.
	f.Close()
	f, err = os.OpenFile(f.Name(), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	err = Write(f, bytes.NewBufferString("ELF-stub"), bytes.NewBufferString("archive contents"), Metadata{ArchiveName: "资料 & test.zip", Directory: "资料 & test"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRoundTrip(t *testing.T) {
	data := fixture(t)
	b, err := Read(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	contents, err := io.ReadAll(b.Archive)
	if err != nil || string(contents) != "archive contents" {
		t.Fatalf("%q: %v", contents, err)
	}
	if b.Metadata.Directory != "资料 & test" {
		t.Fatal(b.Metadata)
	}
}

func TestCorruption(t *testing.T) {
	for _, tc := range []string{"payload", "metadata", "magic", "version", "reserved", "stub-overflow", "archive-overflow", "metadata-overflow", "truncated", "trailing"} {
		t.Run(tc, func(t *testing.T) {
			data := fixture(t)
			footer := data[len(data)-footerSize:]
			switch tc {
			case "payload":
				data[8] ^= 1
			case "metadata":
				data[24] ^= 1
			case "magic":
				footer[0] ^= 1
			case "version":
				footer[8] = 99
			case "reserved":
				footer[12] = 1
			case "stub-overflow":
				binary.LittleEndian.PutUint64(footer[16:24], ^uint64(0))
			case "archive-overflow":
				binary.LittleEndian.PutUint64(footer[24:32], ^uint64(0))
			case "metadata-overflow":
				binary.LittleEndian.PutUint64(footer[32:40], ^uint64(0))
			case "truncated":
				data = data[:len(data)-1]
			case "trailing":
				data = append(data, 0)
			}
			if _, err := Read(bytes.NewReader(data), int64(len(data))); err == nil {
				t.Fatal("accepted damaged bundle")
			}
		})
	}
}

func TestMetadataPaths(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../escape", "/absolute", "a\\b", "C:escape", "nul\x00"} {
		if err := (Metadata{ArchiveName: "test.zip", Directory: name}).validate(); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestDirectory(t *testing.T) {
	for name, want := range map[string]string{"example.zip": "example", "example.tar.gz": "example", "example.TAR.XZ": "example", "裸文件": "裸文件", ".zip": "extracted"} {
		if got := Directory(name); got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}

func FuzzRead(f *testing.F) {
	f.Add([]byte("not a bundle"))
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = Read(bytes.NewReader(data), int64(len(data))) })
}
