// Command enumgen generates C headers from Go enum declarations.
package main

import (
	"flag"
	"log"
	"os"
	"text/template"
)

func main() {
	var ta templateArgs
	flag.StringVar(&ta.Guard, "guard", "", "#include guard name")
	outputFlag := flag.String("out", "", "output filename")
	flag.Parse()
	if flag.NArg() != 1 || *outputFlag == "" {
		log.Fatal("Usage: enumgen -guard=INCLUDE_GUARD -out=output.h path/to/package")
	}
	ta.RecognizeFiles(flag.Arg(0))

	outFile, e := os.Create(*outputFlag)
	if e != nil {
		log.Fatal("cannot open output file", e)
	}
	defer outFile.Close()
	outputTpl.Execute(outFile, ta)
}

type templateArgs struct {
	pkgConsts
	Guard string
}

const outputTemplate = `
{{- /* */ -}}
// <auto-generated> ndn-dpdk/mk/enumgen
#ifndef {{.Guard}}
#define {{.Guard}}
#include <stdbool.h>
{{range .Enums}}
{{- if .Typename}}
typedef enum {{.Typename}} {
{{- else}}
enum {
{{- end}}
{{- range .Definitions}}
  {{.Key}} = {{.Value}},
  {{- end}}
{{- if .Typename}}
} __attribute__((packed)) {{.Typename}};
{{- else}}
};
{{- end}}
{{end}}
#endif // {{.Guard}}
`

var outputTpl = template.Must(template.New("").Parse(outputTemplate))
