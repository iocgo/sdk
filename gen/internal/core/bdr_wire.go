package core

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

func Wire(target string) func(proc *Processor) (ops map[string][]byte) {
	return func(proc *Processor) (ops map[string][]byte) {
		ops = make(map[string][]byte)
		tfs := token.NewFileSet()
		f, err := parser.ParseFile(tfs, target, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}

		var fd *ast.FuncDecl
		imports := make([]Imported, 0)
		pos := 0
		ast.Inspect(f, func(node ast.Node) bool {
			switch expr := node.(type) {
			case *ast.ImportSpec:
				if expr.Path.Value == `"github.com/iocgo/sdk"` {
					return true
				}

				alias := ""
				if expr.Name != nil {
					alias = expr.Name.Name
				}

				if alias == "" || alias == "_" {
					pos++
					alias = "pak" + strconv.Itoa(pos)
					expr.Name.Name = alias
				}

				imports = append(imports, Imported{alias, expr.Path.Value})
			case *ast.FuncDecl:
				if expr.Name.Name != "Injects" {
					return true
				}
				if expr.Type.Params == nil || len(expr.Type.Params.List) != 1 {
					return true
				}

				fd = expr
				return false
			}
			return true
		})

		if fd == nil {
			panic("not found `Injects(container) error` method")
		}

		buf := strings.Builder{}
		line := fmt.Sprintf("//line %s:%d\n", target, tfs.Position(fd.Pos()).Line)
		buf.WriteString(line)
		buf.WriteString("func(container sdk.Container) error {\n")
		for _, ip := range imports {
			buf.WriteString(line)
			buf.WriteString(fmt.Sprintf("\tif err := %s.Injects(container); err != nil {\n", ip.Alias))
			buf.WriteString("\t\treturn err\n")
			buf.WriteString("\t}\n")
		}
		buf.WriteString("return nil }()")
		expr, err := parser.ParseExpr(buf.String())
		if err != nil {
			panic(err)
		}

		field := fd.Type.Params.List[0]
		if field.Names == nil {
			field.Names = []*ast.Ident{{Name: "container", NamePos: field.Pos()}}
		}
		if field.Names[0].Name != "container" {
			field.Names[0].Name = "container"
		}

		fd.Body = assignPos(expr.(*ast.CallExpr).Fun.(*ast.FuncLit).Body, fd.Pos())
		str, err := nodeString(tfs, f)
		if err != nil {
			panic(err)
		}

		_, importPath, err := commandAsImportPath(filepath.Dir(target))
		if err != nil {
			panic(err)
		}

		ops[filepath.Join(tempDir, importPath, filepath.Base(target))] = []byte(str)
		return
	}
}

func assignPos(stmt *ast.BlockStmt, pos token.Pos) *ast.BlockStmt {
	// TODO
	for _, i := range stmt.List {
		switch expr := i.(type) {
		case *ast.IfStmt:
			expr.If = pos
			assignPos(expr.Body, pos)
		case *ast.ReturnStmt:
			expr.Return = pos
		}
	}
	return stmt
}

func nodeString(tfs *token.FileSet, node ast.Node) (str string, err error) {
	var printerCfg = &printer.Config{Tabwidth: 4, Mode: printer.SourcePos}
	var output []byte
	buffer := bytes.NewBuffer(output)
	err = printerCfg.Fprint(buffer, tfs, node)
	if err == nil {
		str = buffer.String()
	}
	return
}
