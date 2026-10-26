// Package extract confines archive writes to the selected destination.
package extract

import (
	"context"
	"fmt"
	"github.com/mholt/archives"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Unarchive extracts regular files and directories. Links and special files are
// refused: they must never redirect a later entry outside the destination.
func Unarchive(source, destination string) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()
	return Reader(f, source, destination)
}

// Reader extracts directly from a bounded archive section without a temporary copy.
func Reader(source io.Reader, name, destination string) error {
	format, reader, err := archives.Identify(context.Background(), name, source)
	if err != nil {
		return err
	}
	extractor, ok := format.(archives.Extractor)
	if !ok {
		return fmt.Errorf("unsupported archive format: %s", name)
	}
	if err = os.MkdirAll(destination, 0755); err != nil {
		return err
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return err
	}
	defer root.Close()
	return extractor.Extract(context.Background(), reader, func(ctx context.Context, entry archives.FileInfo) error {
		name := strings.TrimSuffix(entry.NameInArchive, "/")
		if name == "." && entry.IsDir() {
			return nil
		}
		if !filepath.IsLocal(name) || strings.ContainsAny(name, "\\:") {
			return fmt.Errorf("unsafe archive path: %q", name)
		}
		if entry.LinkTarget != "" || (!entry.IsDir() && !entry.Mode().IsRegular()) {
			return fmt.Errorf("unsupported archive link or special file: %q", name)
		}
		if entry.IsDir() {
			return root.MkdirAll(name, 0755)
		}
		if err := root.MkdirAll(filepath.Dir(name), 0755); err != nil {
			return err
		}
		in, err := entry.Open()
		if err != nil {
			return err
		}
		defer in.Close()
		// Exclusive creation also prevents overwriting an existing hard link.
		out, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, entry.Mode().Perm()&0755)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
