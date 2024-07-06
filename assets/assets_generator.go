// +build

package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
)

func main() {
	gPtr := flag.String("go", "", "")
	wPtr := flag.String("wrapper", "", "")
	flag.Parse()

	templateStr := `
package assets

import (
    "log"
    "os"
    "embed"
    "path/filepath"

    "github.com/mholt/archiver/v3"
)

//go:embed go/{{.GO}}
var AssetsGo embed.FS

func AssetsGoExtract(dst string){
    dstArch := filepath.Join(dst, "{{.GO}}")
    data, err := AssetsGo.ReadFile("go/"+"{{.GO}}")
    if err != nil {
        log.Println(err)
    }
    if err := os.WriteFile(dstArch, data, 0644); err != nil {
        log.Println(err)
    }

	err = archiver.Unarchive(dstArch, filepath.Join(dst, "go"))
	if err != nil {
		log.Println(err)
	}
}

//go:embed wrapper/{{.Wrapper}}
var AssetsWrapper embed.FS

func AssetsWrapperExtract(dst string){
    dstArch := filepath.Join(dst , "{{.Wrapper}}")
    data, err := AssetsWrapper.ReadFile("wrapper/"+"{{.Wrapper}}")
    if err != nil {
        log.Println(err)
    }
    if err := os.WriteFile(dstArch, data, 0644); err != nil {
        log.Println(err)
    }

	err = archiver.Unarchive(dstArch, filepath.Join(dst, "wrapper"))
	if err != nil {
		log.Println(err)
	}
    defer os.RemoveAll(dstArch)
}
`

	tmpl := template.Must(template.New("assets").Parse(templateStr))
	data := struct {
		GO      string
		Wrapper string
	}{
		GO:      *gPtr,
		Wrapper: *wPtr,
	}

	file, err := os.Create("assets.go")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	err = tmpl.Execute(file, data)
	if err != nil {
		panic(err)
	}

	fmt.Println("File generated：assets.go")
}
