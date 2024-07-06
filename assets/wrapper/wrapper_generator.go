package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

func main() {
	filePtr := flag.String("file", "", "file directory")
	flag.Parse()

	templateStr := `
package main

import (
    "io"
    "os"
    "bytes"
	"log"
    "path/filepath"
    _ "embed"

	"github.com/mholt/archiver/v3"
)

//go:embed {{.File}}
var File []byte

func main() {
	tmpDir, err := os.MkdirTemp("", "selfext")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

    extractedFile, err := os.Create(filepath.Join(tmpDir, "{{.File}}"))
    if err != nil {
        log.Fatal(err)
    }
    defer extractedFile.Close()

    _, err = io.Copy(extractedFile, bytes.NewReader(File))
    if err != nil {
        log.Fatal(err)
    }

	err = archiver.Unarchive(filepath.Join(tmpDir, "{{.Src}}"), "{{.Dst}}")
	if err != nil {
		log.Fatal(err)
	}
}
`

	tmpl := template.Must(template.New("file").Parse(templateStr))
	file := *filePtr
	src := filepath.Base(file)
	srcExtension := filepath.Ext(src)
	dst := src[:len(src)-len(srcExtension)]
	data := struct {
		File string
		Src  string
		Dst  string
	}{
		File: *filePtr,
		Src:  src,
		Dst:  dst,
	}

	fd, err := os.Create("wrapper.go")
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	err = tmpl.Execute(fd, data)
	if err != nil {
		panic(err)
	}

	fmt.Println("File Generate：wrapper.go")
}
