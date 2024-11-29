package core

import (
	"bytes"
	"errors"
	"fmt"
	annotations "github.com/bincooo/go-annotation/pkg"
	"github.com/iocgo/sdk/gen/annotation"
	goMeta "github.com/iocgo/sdk/gen/internal/meta"
	"go/ast"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"

	. "github.com/iocgo/sdk/stream"
)

var (
	pxTemplate0 = `package {{ .package }}
//line {{ $.file }}:1
import (
	"reflect"
	"github.com/iocgo/sdk/proxy"
{{- range $import := .imports}}
	{{$import.String}}
{{- end}}
)

{{ .code }}
`
	pxTemplate1 = `
type _alias_{{ replace .name "." "__" }}__ {{ .name }}
type _{{ replace .name "." "__" }}_px__ struct { _alias_{{ replace .name "." "__" }}__ }

func init() {
	proxy.Reg[{{ .name }}](Make_{{ replace .name "." "__" }}_proxy__)
}

func Make_{{ replace .name "." "__" }}_proxy__(instance {{ .name }}) ({{ .name }}, bool) {
	if !proxy.Matched("{{ .scan }}", instance) {
		return instance, false
	}
	return &_{{ replace .name "." "__" }}_px__{instance}, true
}

func __px_elseOf[T any](o any) (t T) {
	if o == nil {
		return
	}
	return o.(T)
}
`
)

func Proxy(proc *Processor) (ops map[string][]byte) {
	ops = make(map[string][]byte)
	for node, converters := range proc.mapping {
		meta := node.Meta()
		for _, convert := range converters {
			if !convert.As("proxy") {
				continue
			}

			imports := make([]Imported, 0)
			var spec *ast.TypeSpec
			n := convert.tag.(annotation.Proxy).Target
			// 定位接口ast语法树
			if idx := strings.LastIndex(n, "."); idx > 0 {
				if meta.PackageName() == n[:idx] {
					n = n[idx+1:]
					spec = panicOnError(goMeta.GetInterfaceInfo(filepath.Join(meta.Dir(), meta.FileName()), n))
				} else {
					packageInfo := panicOnError(goMeta.GetPackageInfo(meta.Dir()))
					dir := panicOnError(packageInfo.FindPackageDirFor(n[:idx]))
					packageInfo = panicOnError(goMeta.GetPackageInfo(dir))
					imports, _ = Import(imports, packageInfo.Name, packageInfo.ImportPath)
					n = packageInfo.Name + "." + n[idx+1:]
					spec = panicOnError(goMeta.GetInterfaceInfo(packageInfo.Dir, n))
				}
			} else {
				packageInfo := panicOnError(goMeta.GetPackageInfo(meta.Dir()))
				dir := panicOnError(packageInfo.FindPackageDirFor(n[:idx]))
				packageInfo = panicOnError(goMeta.GetPackageInfo(dir))
				imports, _ = Import(imports, packageInfo.Name, packageInfo.ImportPath)
				n = packageInfo.Name + "." + n
				spec = panicOnError(goMeta.GetInterfaceInfo(packageInfo.Dir, n))
			}

			scan := convert.tag.(annotation.Proxy).Scan
			instance := panicOnError(template.New(n).
				Funcs(template.FuncMap{
					"replace": strings.ReplaceAll,
				}).
				Parse(pxTemplate1))
			var buf bytes.Buffer
			if err := instance.Execute(&buf, map[string]string{
				"name": n,
				"scan": scan,
			}); err != nil {
				panic(err)
			}

			var igm []string
			if str := convert.tag.(annotation.Proxy).Igm; str != "" {
				igm = strings.Split(str, "&")
			}

			// spec := convert.node.(*ast.TypeSpec)
			methods := spec.Type.(*ast.InterfaceType).Methods
			for _, method := range methods.List {
				eachMethod(convert, n, method, node, igm, &buf, func(alias, importPath string) string {
					imports, alias = Import(imports, alias, importPath)
					return alias
				})
			}

			instance = panicOnError(template.New(n).Parse(pxTemplate0))

			var buf1 bytes.Buffer
			if err := instance.Execute(&buf1, map[string]interface{}{
				"imports": imports,
				"package": meta.PackageName(),
				"code":    buf.String(),
				"file":    filepath.Join(meta.Dir(), meta.FileName()),
			}); err != nil {
				panic(err)
			}

			_, importPath, err := commandAsImportPath(rootPath)
			if err != nil {
				panic(err)
			}

			ops[filepath.Join(tempDir, importPath, ToSnakeCase(n)+"_px.gen.go")] = buf1.Bytes()
		}
	}
	// TODO
	return
}

func eachMethod(
	convert Convertor,
	n string,
	method *ast.Field,
	node annotations.Node,
	igm []string,
	buf *bytes.Buffer,
	importTo func(alias, importPath string) string,
) {
	lookup, meta := node.Lookup(), node.Meta()
	switch expr := method.Type.(type) {
	case *ast.InterfaceType:
		for _, m := range expr.Methods.List {
			eachMethod(convert, n, m, node, igm, buf, importTo)
		}
	case *ast.FuncType:
		if char := method.Names[0].String()[0]; (char >= 'a' && char <= 'z') || char == '_' { // 私有方法？？
			return
		}
		if slices.ContainsFunc(igm, equals(method.Names[0].String())) {
			return
		}

		pos := 1
		argNames := make([]string, 0)
		extractArguments := convert.ExtractArguments(lookup, method)
		args := strings.Join(FlatMap(OfSlice(extractArguments), func(t Argv) []string {
			return Map(OfSlice(t.Names), func(n string) string {
				if n == "" || n == "_" {
					n = "var" + strconv.Itoa(pos)
					pos++
				}
				argNames = append(argNames, n)
				return n + " " + t.String()
			}).ToSlice()
		}).ToSlice(), ", ")

		returnNames := make([]string, 0)
		extractReturns := convert.ExtractReturns(lookup, method)
		returns := strings.Join(FlatMap(OfSlice(extractReturns), func(t Argv) []string {
			return Map(OfSlice(t.Names), func(n string) string {
				if n == "" || n == "_" {
					n = "var" + strconv.Itoa(pos)
					pos++
				}
				returnNames = append(returnNames, n)
				return n + " " + elseOf(t.IsArray, "[]") + elseOf(t.IsPointer, "*") + t.String()
			}).ToSlice()
		}).ToSlice(), ", ")
		if returns != "" {
			returns = "(" + returns + ")"
		}

		line := fmt.Sprintf("//line %s:%d", filepath.Join(meta.Dir(), meta.FileName()), lookup.GetFSet().Position(method.Pos()).Line)
		buf.WriteString(line + "\n")
		buf.WriteString(fmt.Sprintf(`func (obj *_%s_px__) %s(%s) %s {`, strings.ReplaceAll(n, ".", "__"), method.Names[0].String(), args, returns))
		buf.WriteString(fmt.Sprintf(`
					var __px_ctx_ = &proxy.Context{
						Method:   "%s",
						Receiver: reflect.ValueOf(obj._alias_%s__),
						In:       []any{%s},
						Out:      []any{%s},
					}`, method.Names[0].String(), strings.ReplaceAll(n, ".", "__"), strings.Join(argNames, ", "), strings.Join(returnNames, ", ")))

		pos = 0
		args = strings.Join(FlatMap(OfSlice(extractArguments), func(t Argv) []string {
			return Map(OfSlice(t.Names), func(_ string) (str string) {
				str = fmt.Sprintf("__px_ctx_.In[%d].(%s)", pos, elseOf(t.IsArray, "[]")+elseOf(t.IsPointer, "*")+t.Interface.String())
				if !goMeta.IsBaseTyp(t.Interface.String()) {
					packageInfo := panicOnError(goMeta.FindPackageByImports(node.Imports(), t.Interface.Alias()))
					_ = importTo(t.Interface.Alias(), packageInfo.ImportPath)
				}
				pos++
				return
			}).ToSlice()
		}).ToSlice(), ", ")

		pos = 0
		returns = strings.Join(Map(OfSlice(returnNames), func(n string) (str string) {
			str = fmt.Sprintf("__px_ctx_.Out[%d] = %s", pos, n)
			return
		}).ToSlice(), "\n")

		vars := strings.Join(returnNames, ", ")
		if vars != "" {
			vars = vars + " = "
		}

		buf.WriteString("\n" + line)
		buf.WriteString(fmt.Sprintf(`
					__px_ctx_.Do = func() {
						%s
						%sobj._alias_%s__.%s(%s)
						%s
						%s
					}`, line, vars, strings.ReplaceAll(n, ".", "__"), method.Names[0].String(), args, line, returns))

		buf.WriteString("\n" + line)
		buf.WriteString(fmt.Sprintf("\n\t%s(__px_ctx_)\n", convert.GetAstName()))

		pos = 0
		returns = strings.Join(FlatMap(OfSlice(extractReturns), func(t Argv) []string {
			return Map(OfSlice(t.Names), func(_ string) (str string) {
				str = fmt.Sprintf("__px_elseOf[%s](__px_ctx_.Out[%d])",
					elseOf(t.IsArray, "[]")+elseOf(t.IsPointer, "*")+t.Interface.String(),
					pos,
				)
				if !goMeta.IsBaseTyp(t.Interface.String()) {
					packageInfo, err := goMeta.FindPackageByImports(node.Imports(), t.Interface.Alias())
					if err != nil {
						panic(errors.Join(err, fmt.Errorf(`func %s(?? %s:  is not found`, method.Names[0].String(), t.Interface.String())))
					}
					_ = importTo(t.Interface.Alias(), packageInfo.ImportPath)
				}
				pos++
				return
			}).ToSlice()
		}).ToSlice(), ", ")
		if returns != "" {
			returns = line + "\nreturn " + returns
		}

		buf.WriteString(fmt.Sprintf("\n\t %s }\n\n", returns))
	}
}

func equals(value string) func(value string) bool {
	return func(ig string) bool {
		ig = strings.TrimSpace(ig)
		if len(ig) == 0 || value == ig {
			return true
		}

		not := false
		if ig[0] == '!' {
			not = true
			ig = strings.TrimSpace(ig[1:])
		}

		// ! ( condition_1 | condition_2 )
		if igLen := len(ig); igLen > 1 && ig[0] == '(' && ig[igLen-1] == ')' {
			slice := strings.Split(ig[1:igLen-1], "|")
			for _, item := range slice {
				item = strings.TrimSpace(item)
				if !panicOnError(filepath.Match(item, value)) {
					continue
				}
				if not {
					return false
				}
			}
			return not
		}

		if value == ig {
			return !not
		}

		matched := panicOnError(filepath.Match(ig, value))
		return not != matched
	}
}

func elseOf[T any](condition bool, t T) (zero T) {
	if condition {
		return t
	}
	return
}
