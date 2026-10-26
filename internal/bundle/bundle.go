// Package bundle implements the versioned selfext container:
// executable | archive | JSON metadata | 72-byte footer | optional Mach-O signature.
package bundle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const footerSize = 72
const maxMetadata = 64 * 1024

var magic = [8]byte{'S', 'E', 'L', 'F', 'E', 'X', 'T', 0}

type Metadata struct {
	ArchiveName string `json:"archive_name"`
	Directory   string `json:"directory"`
}

type Bundle struct {
	Metadata Metadata
	Archive  *io.SectionReader
}

func validName(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, "/\\:\x00")
}

func (m Metadata) validate() error {
	if !validName(m.ArchiveName) || !validName(m.Directory) {
		return errors.New("invalid archive name or extraction directory")
	}
	return nil
}

// Directory removes common compound archive extensions as well as single ones.
func Directory(name string) string {
	lower := strings.ToLower(name)
	for _, ext := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst", ".tar.lz4", ".tar.br", ".tar.lz"} {
		if strings.HasSuffix(lower, ext) {
			name = name[:len(name)-len(ext)]
			if validName(name) {
				return name
			}
			return "extracted"
		}
	}
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if !validName(name) {
		return "extracted"
	}
	return name
}

// Write streams the archive into an open, empty, read/write output file.
// The caller must remove the output on failure.
func Write(out *os.File, stub io.Reader, archive io.Reader, m Metadata) error {
	if err := m.validate(); err != nil {
		return err
	}
	meta, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if len(meta) > maxMetadata-16 {
		return errors.New("metadata too large")
	}
	stubSize, err := io.Copy(out, stub)
	if err != nil {
		return fmt.Errorf("copy extractor: %w", err)
	}
	layout, err := inspectMachO(out)
	if err != nil {
		return err
	}
	if layout != nil && layout.sigOffset != 0 {
		if layout.sigOffset+layout.sigSize != stubSize {
			return errors.New("extractor signature is not at end of file")
		}
		stubSize = layout.sigOffset
		if err := out.Truncate(stubSize); err != nil {
			return err
		}
		if _, err := out.Seek(stubSize, io.SeekStart); err != nil {
			return err
		}
	}
	hash := sha256.New()
	archiveSize, err := io.Copy(io.MultiWriter(out, hash), archive)
	if err != nil {
		return fmt.Errorf("copy archive: %w", err)
	}
	// Keep the future Mach-O signature 16-byte aligned. JSON permits whitespace.
	padding := (16 - (stubSize+archiveSize+int64(len(meta))+footerSize)%16) % 16
	meta = append(meta, bytes.Repeat([]byte{' '}, int(padding))...)
	if _, err := io.MultiWriter(out, hash).Write(meta); err != nil {
		return err
	}
	var footer [footerSize]byte
	copy(footer[:8], magic[:])
	binary.LittleEndian.PutUint32(footer[8:12], 1)
	binary.LittleEndian.PutUint64(footer[16:24], uint64(stubSize))
	binary.LittleEndian.PutUint64(footer[24:32], uint64(archiveSize))
	binary.LittleEndian.PutUint64(footer[32:40], uint64(len(meta)))
	copy(footer[40:], hash.Sum(nil))
	if _, err := out.Write(footer[:]); err != nil {
		return err
	}
	if layout != nil && layout.sigOffset != 0 {
		return resignMachO(out, layout)
	}
	return nil
}

// Read validates boundaries and the archive/metadata checksum before extraction.
// The checksum detects damage; it does not authenticate the publisher.
func Read(r io.ReaderAt, size int64) (*Bundle, error) {
	end := size
	layout, err := inspectMachO(r)
	if err != nil {
		return nil, err
	}
	if layout != nil && layout.sigOffset != 0 {
		if layout.sigOffset > size || layout.sigSize != size-layout.sigOffset {
			return nil, errors.New("invalid Mach-O signature bounds")
		}
		end = layout.sigOffset
	}
	if end < footerSize {
		return nil, errors.New("missing selfext footer")
	}
	var footer [footerSize]byte
	if _, err := r.ReadAt(footer[:], end-footerSize); err != nil {
		return nil, err
	}
	if !bytes.Equal(footer[:8], magic[:]) {
		return nil, errors.New("missing selfext footer")
	}
	if binary.LittleEndian.Uint32(footer[8:12]) != 1 || binary.LittleEndian.Uint32(footer[12:16]) != 0 {
		return nil, errors.New("unsupported selfext format version")
	}
	stubSize := binary.LittleEndian.Uint64(footer[16:24])
	archiveSize := binary.LittleEndian.Uint64(footer[24:32])
	metaSize := binary.LittleEndian.Uint64(footer[32:40])
	available := uint64(end - footerSize)
	// Subtract only after checking, so hostile lengths cannot wrap around.
	if stubSize == 0 || stubSize > available {
		return nil, errors.New("invalid extractor bounds")
	}
	available -= stubSize
	if archiveSize == 0 || archiveSize > available {
		return nil, errors.New("invalid archive bounds")
	}
	available -= archiveSize
	if metaSize == 0 || metaSize > maxMetadata || metaSize != available {
		return nil, errors.New("invalid metadata bounds")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.NewSectionReader(r, int64(stubSize), int64(archiveSize+metaSize))); err != nil {
		return nil, err
	}
	if !bytes.Equal(hash.Sum(nil), footer[40:]) {
		return nil, errors.New("archive checksum mismatch")
	}
	meta := make([]byte, int(metaSize))
	if _, err := r.ReadAt(meta, int64(stubSize+archiveSize)); err != nil {
		return nil, err
	}
	var m Metadata
	if err := json.Unmarshal(meta, &m); err != nil {
		return nil, err
	}
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &Bundle{Metadata: m, Archive: io.NewSectionReader(r, int64(stubSize), int64(archiveSize))}, nil
}
