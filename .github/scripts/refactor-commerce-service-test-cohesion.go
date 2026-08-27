package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	pathpkg "path"
	"sort"
	"strconv"
	"strings"
)

const sourcePath = "server/internal/modules/commerce/service_test.go"

type group struct {
	file  string
	funcs []*ast.FuncDecl
}

type edit struct {
	start int
	end   int
	text  []byte
}

func main() {
	src, err := os.ReadFile(sourcePath)
	must(err)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, sourcePath, src, parser.ParseComments)
	must(err)
	tf := fset.File(f.Pos())

	groups := map[string]*group{
		"checkout": {file: "server/internal/modules/commerce/service_checkout_test.go"},
		"orders":   {file: "server/internal/modules/commerce/service_orders_test.go"},
		"returns":  {file: "server/internal/modules/commerce/service_returns_test.go"},
		"restock":  {file: "server/internal/modules/commerce/service_restock_test.go"},
	}
	moved := map[*ast.FuncDecl]bool{}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fd.Name.Name, "Test") {
			continue
		}
		name := fd.Name.Name
		key := classify(name)
		if key == "" {
			continue
		}
		groups[key].funcs = append(groups[key].funcs, fd)
		moved[fd] = true
	}
	for name, g := range groups {
		if len(g.funcs) == 0 {
			panic("no tests classified for " + name)
		}
	}

	imports, importDecl := importSpecs(f)
	if importDecl == nil {
		panic("service_test.go has no import declaration")
	}

	// Rewrite service_test.go by removing only moved Test* declarations and
	// shrinking its import block to packages still referenced by kept decls.
	keptDecls := make([]ast.Decl, 0, len(f.Decls))
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && moved[fd] {
			continue
		}
		keptDecls = append(keptDecls, d)
	}
	usedKept := usedIdentifiers(keptDecls)
	edits := []edit{{
		start: tf.Offset(importDecl.Pos()),
		end:   tf.Offset(importDecl.End()),
		text:  []byte(renderImports(imports, usedKept)),
	}}
	for fd := range moved {
		start := fd.Pos()
		if fd.Doc != nil {
			start = fd.Doc.Pos()
		}
		edits = append(edits, edit{start: tf.Offset(start), end: tf.Offset(fd.End())})
	}
	out := append([]byte(nil), src...)
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	for _, e := range edits {
		out = append(out[:e.start], append(e.text, out[e.end:]...)...)
	}
	formatted, err := format.Source(out)
	must(err)
	must(os.WriteFile(sourcePath, formatted, 0o644))

	// Build dedicated test files from exact AST byte ranges. Imports are
	// derived from identifier usage in the moved functions, so no speculative
	// or unused package imports are introduced.
	for _, key := range []string{"checkout", "orders", "returns", "restock"} {
		g := groups[key]
		used := usedFuncIdentifiers(g.funcs)
		var b bytes.Buffer
		b.WriteString("package commerce\n\n")
		b.WriteString(renderImports(imports, used))
		b.WriteString("\n\n")
		for i, fd := range g.funcs {
			start := fd.Pos()
			if fd.Doc != nil {
				start = fd.Doc.Pos()
			}
			b.Write(bytes.TrimSpace(src[tf.Offset(start):tf.Offset(fd.End())]))
			b.WriteString("\n")
			if i != len(g.funcs)-1 {
				b.WriteString("\n")
			}
		}
		formatted, err := format.Source(b.Bytes())
		must(err)
		must(os.WriteFile(g.file, formatted, 0o644))
		fmt.Printf("%s: %d tests -> %s\n", key, len(g.funcs), g.file)
	}
}

func classify(name string) string {
	if strings.Contains(name, "Restock") {
		return "restock"
	}
	if strings.Contains(name, "Return") {
		return "returns"
	}
	for _, s := range []string{"CreateOrder", "Quote", "Idempotenc", "Fingerprint", "Payload", "Overflow", "OrderInput", "ResolveItems"} {
		if strings.Contains(name, s) {
			return "checkout"
		}
	}
	for _, s := range []string{"Order", "Guest", "Member", "Mask"} {
		if strings.Contains(name, s) {
			return "orders"
		}
	}
	return ""
}

func importSpecs(f *ast.File) ([]*ast.ImportSpec, *ast.GenDecl) {
	var specs []*ast.ImportSpec
	var decl *ast.GenDecl
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		if decl == nil {
			decl = gd
		}
		for _, s := range gd.Specs {
			specs = append(specs, s.(*ast.ImportSpec))
		}
	}
	return specs, decl
}

func usedFuncIdentifiers(funcs []*ast.FuncDecl) map[string]bool {
	decls := make([]ast.Decl, len(funcs))
	for i, f := range funcs {
		decls[i] = f
	}
	return usedIdentifiers(decls)
}

func usedIdentifiers(decls []ast.Decl) map[string]bool {
	used := map[string]bool{}
	for _, d := range decls {
		ast.Inspect(d, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if ok {
				used[id.Name] = true
			}
			return true
		})
	}
	return used
}

func renderImports(specs []*ast.ImportSpec, used map[string]bool) string {
	var lines []string
	for _, s := range specs {
		path, err := strconv.Unquote(s.Path.Value)
		must(err)
		name := pathpkg.Base(path)
		if s.Name != nil {
			name = s.Name.Name
		}
		if name != "_" && name != "." && !used[name] {
			continue
		}
		line := "\t"
		if s.Name != nil {
			line += s.Name.Name + " "
		}
		line += strconv.Quote(path)
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	return "import (\n" + strings.Join(lines, "\n") + "\n)"
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
