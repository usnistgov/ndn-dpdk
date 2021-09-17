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

	var collect []string
	for _, spec := range decl.Specs {
		vspec := spec.(*ast.ValueSpec)
		if len(vspec.Names) != 1 || vspec.Names[0].Name != "_" || len(vspec.Values) != 1 {
			continue
		}
		tokens := strings.Split(strings.Trim(pc.nodeToString(vspec.Values[0]), "\""), ":")
		if tokens[0] != "enumgen" {
			continue
		}
		collect = tokens
		break
	}
	if len(collect) == 0 {
		return
	}

	typename, prependPrefix, trimPrefix := "", "", ""
	switch len(collect) {
	case 4:
		trimPrefix = collect[3]
		fallthrough
	case 3:
		prependPrefix = collect[2]
		fallthrough
	case 2:
		typename = collect[1]
	}

	enum := pc.Enums[typename]
	if enum == nil {
		enum = new(enumDecl)
		enum.Typename = typename
		pc.Enums[typename] = enum
	}
	rename := func(name string) string { return prependPrefix + strings.TrimPrefix(name, trimPrefix) }
	pc.collectEnum(enum, decl, rename)
}

func (pc *pkgConsts) collectEnum(enum *enumDecl, decl *ast.GenDecl, rename func(string) string) {
	skipUntil := -1
	for i, spec := range decl.Specs {
		vspec := spec.(*ast.ValueSpec)
		if len(vspec.Names) != 1 || len(vspec.Values) > 1 || i <= skipUntil {
			continue
		}

		name := vspec.Names[0].Name
		if name == "_" {
			if len(vspec.Values) == 1 {
				str := strings.Trim(pc.nodeToString(vspec.Values[0]), "\"")
				if strings.HasPrefix(str, "enumgen+") {
					nSkip, _ := strconv.Atoi(str[8:])
					skipUntil = i + nSkip
				}
			}
			continue
		}
		name = rename(name)

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
