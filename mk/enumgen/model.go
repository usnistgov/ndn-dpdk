package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

type kv struct {
	Key   string
	Value string
}

type enumDecl struct {
	Typename    string
	Definitions []kv
}

type pkgConsts struct {
	fset  *token.FileSet
	Enums map[string]*enumDecl
}

func (pc *pkgConsts) RecognizeFiles(path string) {
	pc.fset = token.NewFileSet()
	pc.Enums = make(map[string]*enumDecl)

	pkgs, e := parser.ParseDir(pc.fset, path, nil, 0)
	if e != nil {
		panic(e)
	}
	for _, pkg := range pkgs {
		file := ast.MergePackageFiles(pkg, 0)
		pc.recognizeFile(file)
	}
}

func (pc *pkgConsts) recognizeFile(file *ast.File) {
	for _, decl := range file.Decls {
		gdecl, ok := decl.(*ast.GenDecl)
		if !ok || gdecl.Tok != token.CONST {
			continue
		}
		pc.recognizeConstDecl(gdecl)
	}
}

func (pc *pkgConsts) recognizeConstDecl(decl *ast.GenDecl) {
	if len(decl.Specs) == 0 {
		return
	}

	wantCollect := false
	typename := ""

	for _, spec := range decl.Specs {
		vspec := spec.(*ast.ValueSpec)
		if len(vspec.Names) != 1 || vspec.Names[0].Name != "_" || len(vspec.Values) != 1 {
			continue
		}
		value := pc.nodeToString(vspec.Values[0])
		switch {
		case value == "\"enumgen\"":
			wantCollect = true
		case strings.HasPrefix(value, "\"enumgen:") && strings.HasSuffix(value, "\""):
			wantCollect = true
			typename = value[9 : len(value)-1]
		}
		break
	}

	if wantCollect {
		enum := pc.Enums[typename]
		if enum == nil {
			enum = new(enumDecl)
			enum.Typename = typename
			pc.Enums[typename] = enum
		}
		pc.collectEnum(enum, decl)
	}
}

func (pc *pkgConsts) collectEnum(enum *enumDecl, decl *ast.GenDecl) {
	for i, spec := range decl.Specs {
		vspec := spec.(*ast.ValueSpec)
		if len(vspec.Names) != 1 || len(vspec.Values) > 1 {
			continue
		}

		name := vspec.Names[0].Name
		if name == "_" {
			continue
		}

		value := strconv.Itoa(i) // iota
		if len(vspec.Values) == 1 {
			value = pc.nodeToString(vspec.Values[0])
			if value == "iota" {
				value = "0"
			}
		}

		enum.Definitions = append(enum.Definitions, kv{name, value})
	}
}

func (pc *pkgConsts) nodeToString(node interface{}) string {
	var buf bytes.Buffer
	format.Node(&buf, pc.fset, node)
	return strings.ReplaceAll(buf.String(), "\n\t", " ")
}
