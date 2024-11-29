package core

import (
	"fmt"
	annotation "github.com/bincooo/go-annotation/pkg"
	gen "github.com/iocgo/sdk/gen/annotation"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"

	. "github.com/iocgo/sdk/stream"
)

type GoAst int

const (
	_ GoAst = iota
	GoFunc
	GoStruct
	GoInterface
)

type Interface string

type Argv struct {
	alias      string
	ImportPath string
	Names      []string
	Interface  Interface
	IsPointer  bool
	IsArray    bool
}

func (i Interface) Alias() string {
	s := strings.Split(string(i), ".")
	return Or(len(s) > 1, s[0], "")
}

func (i Interface) Ext() string {
	s := strings.Split(string(i), ".")
	return Or(len(s) > 1, strings.Join(s[1:], "."), string(i))
}

func (i Interface) String() string {
	return string(i)
}

func (re *Argv) Alias(n string) {
	re.alias = n
}

func (re *Argv) String() string {
	ptr := Or(re.IsPointer, "*", "")
	if re.alias == "" {
		return ptr + re.Interface.String()
	}
	return ptr + re.alias + "." + re.Interface.String()
}

type Convertor struct {
	tag gen.M

	enum GoAst
	node ast.Node

	alias      string
	importPath string
}

func newConvertor(tag gen.M, node ast.Node, importPath string) Convertor {
	var enum GoAst = 0
	switch node.(type) {
	case *ast.TypeSpec:
		enum = GoInterface
	case *ast.StructType:
		enum = GoStruct
	case *ast.FuncDecl:
		enum = GoFunc
	}

	return Convertor{
		tag:        tag,
		enum:       enum,
		node:       node,
		importPath: importPath,
	}
}

func (convert *Convertor) Alias(n string) {
	convert.alias = n
}

func (convert *Convertor) GetAstName() (n string) {
	if convert.Is(GoFunc) {
		fd := convert.node.(*ast.FuncDecl)
		if convert.alias == "" {
			return fd.Name.Name
		}
		return convert.alias + "." + fd.Name.Name
	}

	if convert.Is(GoInterface) {
		spec := convert.node.(*ast.TypeSpec)
		if convert.alias == "" {
			return spec.Name.Name
		}
		return convert.alias + "." + spec.Name.Name
	}

	// TODO -
	return
}

func (convert *Convertor) ExtractArguments(lookup annotation.Lookup, node ast.Node) (args []Argv) {
	switch expr := node.(type) {
	case *ast.FuncDecl:
		results := expr.Type.Params
		args = convert.extractArgs(lookup, results)
		return
	case *ast.Field:
		results := expr.Type.(*ast.FuncType).Params
		args = convert.extractArgs(lookup, results)
	}
	return
}

func (convert *Convertor) ExtractReturns(lookup annotation.Lookup, node ast.Node) (args []Argv) {
	switch expr := node.(type) {
	case *ast.FuncDecl:
		results := expr.Type.Results
		args = convert.extractArgs(lookup, results)
		return
	case *ast.Field:
		results := expr.Type.(*ast.FuncType).Results
		args = convert.extractArgs(lookup, results)
	}
	return
}

func (convert *Convertor) extractArgs(lookup annotation.Lookup, results *ast.FieldList) (args []Argv) {
	if results == nil {
		return
	}

	for _, re := range results.List {
		var isPointer, isArray bool
		var interfaceName string

		switch expr := re.Type.(type) {
		case *ast.StarExpr:
			isPointer = true
			interfaceName = convert.parseInterfaceName(expr.X)
		case *ast.SelectorExpr:
			interfaceName = convert.parseInterfaceName(expr)
		case *ast.ArrayType:
			isArray = true
			interfaceName = convert.parseInterfaceName(expr.Elt)
		default:
			interfaceName = re.Type.(*ast.Ident).Name
		}

		importPath := ""
		if slice := strings.Split(interfaceName, "."); len(slice) > 1 {
			result, ok := lookup.FindImportByAlias(slice[0])
			if ok {
				importPath = result
			}
		}

		names := Map(OfSlice(re.Names), func(id *ast.Ident) string {
			return id.Name
		}).ToSlice()
		if len(names) == 0 {
			names = []string{""}
		}

		args = append(args, Argv{
			ImportPath: importPath,
			Interface:  Interface(interfaceName),
			Names:      names,
			IsPointer:  isPointer,
			IsArray:    isArray,
		})
	}
	return args
}

func (convert *Convertor) ImportPath() string {
	return convert.importPath
}

func (convert *Convertor) Is(enum GoAst) bool {
	if convert.enum == 0 {
		return false
	}

	return convert.enum == enum
}

func (convert *Convertor) As(name string) bool {
	return convert.tag.Name() == name
}

func (convert *Convertor) String() (str string, err error) {
	var buf strings.Builder
	err = printer.Fprint(&buf, token.NewFileSet(), convert.node)
	if err == nil {
		str = buf.String()
	}
	return
}

func (convert *Convertor) parseInterfaceName(expr ast.Expr) string {
	switch ex := expr.(type) {
	case *ast.Ident:
		return ex.Name
	case *ast.SelectorExpr:
		return ex.X.(*ast.Ident).Name + "." + ex.Sel.Name
	case *ast.IndexExpr:
		t := parseT(ex)
		return convert.parseInterfaceName(ex.X) + elseOf(t != "", "["+t+"]")
	}
	panic("parse error")
}

func parseT(expr *ast.IndexExpr) string {
	var genericParams []string
	switch param := expr.Index.(type) {
	case *ast.Ident:
		genericParams = append(genericParams, param.Name)
	// case *ast.BinaryExpr:
	// 	genericParams = append(genericParams, extractBinaryExprString(param))
	case *ast.SelectorExpr:
		genericParams = append(genericParams,
			fmt.Sprintf("%s.%s", param.X.(*ast.Ident).Name, param.Sel.Name))
	case *ast.InterfaceType:
		genericParams = append(genericParams, "interface{}")
	}
	return strings.Join(genericParams, ", ")
}
