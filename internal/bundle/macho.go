package bundle

import (
	"debug/macho"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"

	"github.com/gzj/selfext/internal/codesign"
)

type machLayout struct {
	order                          binary.ByteOrder
	sigCommand, sigOffset, sigSize int64
	linkCommand                    int64
	link, text                     *macho.Segment
}

// Only thin 64-bit Mach-O binaries are emitted by our build matrix.
func inspectMachO(r io.ReaderAt) (*machLayout, error) {
	var magic [4]byte
	if _, err := r.ReadAt(magic[:], 0); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint32(magic[:]) != macho.Magic64 {
		return nil, nil
	}
	m, err := macho.NewFile(r)
	if err != nil {
		return nil, err
	}
	defer m.Close()
	layout := &machLayout{order: m.ByteOrder}
	offset := int64(32)
	for _, load := range m.Loads {
		raw := load.Raw()
		if len(raw) < 8 {
			return nil, errors.New("invalid Mach-O load command")
		}
		if m.ByteOrder.Uint32(raw) == codesign.LC_CODE_SIGNATURE {
			if len(raw) != 16 || layout.sigOffset != 0 {
				return nil, errors.New("invalid Mach-O signature command")
			}
			layout.sigCommand = offset
			layout.sigOffset = int64(m.ByteOrder.Uint32(raw[8:12]))
			layout.sigSize = int64(m.ByteOrder.Uint32(raw[12:16]))
		}
		if seg, ok := load.(*macho.Segment); ok {
			if seg.Name == "__LINKEDIT" {
				layout.link = seg
				layout.linkCommand = offset
			}
			if seg.Name == "__TEXT" {
				layout.text = seg
			}
		}
		offset += int64(len(raw))
	}
	if layout.sigOffset != 0 && (layout.link == nil || layout.text == nil) {
		return nil, errors.New("missing Mach-O segments")
	}
	return layout, nil
}

// Appending bytes invalidates Go's ARM64 ad-hoc signature. Move the signature
// after the payload, extend __LINKEDIT, and hash the final file on any host OS.
func resignMachO(f *os.File, m *machLayout) error {
	end, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	sigSize := codesign.Size(end, "selfext")
	// LC_CODE_SIGNATURE offsets/sizes are uint32, even in a 64-bit Mach-O file.
	if end+sigSize > math.MaxUint32 {
		return errors.New("signed macOS output must be smaller than 4 GiB")
	}
	if uint64(end) < m.link.Offset {
		return errors.New("invalid Mach-O linkedit offset")
	}
	var buf [8]byte
	m.order.PutUint32(buf[:4], uint32(end))
	m.order.PutUint32(buf[4:], uint32(sigSize))
	if _, err := f.WriteAt(buf[:], m.sigCommand+8); err != nil {
		return err
	}
	segmentSize := uint64(end+sigSize) - m.link.Offset
	m.order.PutUint64(buf[:], segmentSize)
	if _, err := f.WriteAt(buf[:], m.linkCommand+32); err != nil {
		return err
	}
	if _, err := f.WriteAt(buf[:], m.linkCommand+48); err != nil {
		return err
	}
	signature := make([]byte, int(sigSize))
	if err := codesign.Sign(signature, io.NewSectionReader(f, 0, end), "selfext", end, int64(m.text.Offset), int64(m.text.Filesz), true); err != nil {
		return err
	}
	_, err = f.Write(signature)
	return err
}
