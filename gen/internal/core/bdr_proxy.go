package core

import (
	"bytes"
	"fmt"
	"github.com/iocgo/sdk/gen/annotation"
	goMeta "github.com/iocgo/sdk/gen/internal/meta"
	"go/ast"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	. "github.com/iocgo/sdk/stream"
)

var (
	pxTemplate0 = `package {{ .package }}

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
type _alias_{{ .name }}__ {{ .name }}
type _{{ replace .name "." "__" }}_px__ struct {
	_alias_{{ .name }}__
}

func init() {
	proxy.Reg[{{ .name }}](Make_{{ replace .name "." "__" }}_proxy__)
}

func Make_{{ replace .name "." "__" }}_proxy__(instance {{ .name }}) {{ .name }} {
	return &_{{ replace .name "." "__" }}_px__{instance}
}

func __px_if_or[T any](o any) (t T) {
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
		lookup := node.Lookup()
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

			instance := panicOnError(template.New(n).
				Funcs(template.FuncMap{
					"replace": strings.ReplaceAll,
				}).
				Parse(pxTemplate1))
			var buf bytes.Buffer
			if err := instance.Execute(&buf, map[string]string{
				"name": n,
			}); err != nil {
				panic(err)
			}

			var (
				pos = 1
			)
			var eachMethod func(method *ast.Field)
			eachMethod = func(method *ast.Field) {
				switch expr := method.Type.(type) {
				case *ast.InterfaceType:
					for _, m := range expr.Methods.List {
						eachMethod(m)
					}
				case *ast.FuncType:
					if char := method.Names[0].String()[0]; (char >= 'a' && char <= 'z') || char == '_' { // 私有方法？？
						return
					}

					argNames := make([]string, 0)
					extractArguments := convert.ExtractArguments(node.Lookup(), method)
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
					extractReturns := convert.ExtractReturns(node.Lookup(), method)
					returns := strings.Join(FlatMap(OfSlice(extractReturns), func(t Argv) []string {
						return Map(OfSlice(t.Names), func(n string) string {
							if n == "" || n == "_" {
								n = "var" + strconv.Itoa(pos)
								pos++
							}
							returnNames = append(returnNames, n)
							return n + " " + t.String()
						}).ToSlice()
					}).ToSlice(), ", ")
					if returns != "" {
						returns = "(" + returns + ")"
					}

					line := fmt.Sprintf("//line %s:%d", filepath.Join(meta.Dir(), meta.FileName()), lookup.GetFSet().Position(method.Pos()).Line)
					buf.WriteString(line + "\n")
					buf.WriteString(fmt.Sprintf(`func (obj *_%s_px__) %s(%s) %s {`, strings.ReplaceAll(n, ".", "__"), method.Names[0].String(), args, returns))
					buf.WriteString(fmt.Sprintf(`
					var ctx = &proxy.Context{
						Name:     "%s",
						Receiver: reflect.ValueOf(obj._alias_%s__),
						In:       []any{%s},
						Out:      []any{%s},
					}`, method.Names[0].String(), n, strings.Join(argNames, ", "), strings.Join(returnNames, ", ")))

					pos = 0
					args = strings.Join(FlatMap(OfSlice(extractArguments), func(t Argv) []string {
						return Map(OfSlice(t.Names), func(_ string) (str string) {
							str = fmt.Sprintf("ctx.In[%d].(%s)", pos, t.Interface.String())
							if !goMeta.IsBaseTyp(t.Interface.String()) {
								packageInfo := panicOnError(goMeta.FindPackageByImports(node.Imports(), t.Interface.Alias()))
								imports, _ = Import(imports, t.Interface.Alias(), packageInfo.ImportPath)
							}
							pos++
							return
						}).ToSlice()
					}).ToSlice(), ", ")

					pos = 0
					returns = strings.Join(Map(OfSlice(returnNames), func(n string) (str string) {
						str = fmt.Sprintf("ctx.Out[%d] = %s", pos, n)
						return
					}).ToSlice(), "\n")

					vars := strings.Join(returnNames, ", ")
					if vars != "" {
						vars = vars + " = "
					}

					shortN := n
					if idx := strings.LastIndex(n, "."); idx > 0 {
						shortN = n[idx+1:]
					}

					buf.WriteString("\n" + line)
					buf.WriteString(fmt.Sprintf(`
					ctx.Do = func() {
						%s
						%sobj._alias_%s__.%s(%s)
						%s
						%s
					}`, line, vars, shortN, method.Names[0].String(), args, line, returns))

					scan := convert.tag.(annotation.Proxy).Scan
					if scan != "" {
						buf.WriteString("\n" + line)
						buf.WriteString(fmt.Sprintf("\n\tif !proxy.Matched(`%s`, obj._alias_%s__) {", scan, shortN))
						if len(extractReturns) > 0 {
							buf.WriteString(fmt.Sprintf("\n\treturn obj._alias_%s__.%s(%s)\n}", shortN, method.Names[0].String(), args))
						} else {
							buf.WriteString("}")
						}
					}
					buf.WriteString("\n" + line)
					buf.WriteString(fmt.Sprintf("\n\t%s(ctx)\n", convert.GetAstName()))

					pos = 0
					returns = strings.Join(FlatMap(OfSlice(extractReturns), func(t Argv) []string {
						return Map(OfSlice(t.Names), func(_ string) (str string) {
							str = fmt.Sprintf("__px_if_or[%s](ctx.Out[%d])", t.Interface.String(), pos)
							if !goMeta.IsBaseTyp(t.Interface.String()) {
								packageInfo := panicOnError(goMeta.FindPackageByImports(node.Imports(), t.Interface.Alias()))
								imports, _ = Import(imports, t.Interface.Alias(), packageInfo.ImportPath)
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

			// spec := convert.node.(*ast.TypeSpec)
			methods := spec.Type.(*ast.InterfaceType).Methods
			for _, method := range methods.List {
				eachMethod(method)
			}

			instance = panicOnError(template.New(n).Parse(pxTemplate0))

			var buf1 bytes.Buffer
			if err := instance.Execute(&buf1, map[string]interface{}{
				"imports": imports,
				"package": node.Meta().PackageName(),
				"code":    buf.String(),
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
