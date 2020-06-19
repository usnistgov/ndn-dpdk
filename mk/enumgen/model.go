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
	fset   *token.FileSet
	Enums  map[string]*enumDecl
	Consts []kv
}

func (pc *pkgConsts) Init(enumList []string) {
	pc.Enums = make(map[string]*enumDecl)
	for _, typename := range enumList {
		pc.Enums[typename] = &enumDecl{typename, nil}
	}
}

func (pc *pkgConsts) RecognizeFiles(path string) {
	pc.fset = token.NewFileSet()
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
	vspec := decl.Specs[0].(*ast.ValueSpec)

	if typeIdent, ok := vspec.Type.(*ast.Ident); ok {
		typename := typeIdent.String()
		if enum, ok := pc.Enums[typename]; ok {
			pc.collectEnum(enum, decl)
			return
		}
	}

	if len(vspec.Values) == 1 && pc.nodeToString(vspec.Values[0]) == "\"enumgen\"" {
		pc.collectConst(decl)
	}
}

func (pc *pkgConsts) collectEnum(enum *enumDecl, decl *ast.GenDecl) {
	for i, spec := range decl.Specs {
		vspec := spec.(*ast.ValueSpec)
		if len(vspec.Names) != 1 || len(vspec.Values) > 1 {
			continue
		}

		name := vspec.Names[0].String()
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

func (pc *pkgConsts) collectConst(decl *ast.GenDecl) {
	for i, spec := range decl.Specs {
		if i == 0 {
			continue
		}
		vspec := spec.(*ast.ValueSpec)
		if len(vspec.Names) != 1 || len(vspec.Values) != 1 {
			continue
		}

		name := vspec.Names[0].String()
		value := pc.nodeToString(vspec.Values[0])
		pc.Consts = append(pc.Consts, kv{name, value})
	}
}

func (pc *pkgConsts) nodeToString(node interface{}) string {
	var buf bytes.Buffer
	format.Node(&buf, pc.fset, node)
	return strings.ReplaceAll(buf.String(), "\n\t", " ")
}
