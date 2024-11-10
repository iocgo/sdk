package core

import (
	"bytes"
	"fmt"
	annotations "github.com/iocgo/sdk/gen/annotation"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	. "github.com/iocgo/sdk/stream"
)

type Imported struct {
	Alias      string
	ImportPath string
}

var (
	iocTemplate = `package {{ .package }}

import (
	"github.com/iocgo/sdk"
{{- range $import := .imports}}
	{{$import.String}}
{{- end}}
)

func Injects(container *sdk.Container) error {
	// Registered container
	//
{{- range $code := .codes}}
	{{$code}}
{{ end }}

	return nil
}
`
)

func (ip Imported) String() string {
	if len(ip.Alias) == 0 {
		return fmt.Sprintf(`"%s"`, ip.ImportPath)
	}
	return fmt.Sprintf(`%s "%s"`, ip.Alias, ip.ImportPath)
}

func Inject(proc *Processor) (ops map[string][]byte) {
	var (
		pkg     = ""
		imports []Imported
		codes,
		activated []string
	)

	for node, convertors := range proc.mapping {
		lookup := node.Lookup()
		if pkg == "" {
			pkg = node.Meta().PackageName()
		}

		for _, convert := range convertors {
			if !convert.As("inject") {
				continue
			}

			// validate
			returns := convert.ExtractReturns(node.Lookup(), convert.node)
			{
				types := FlatMap(OfSlice(returns),
					func(re Argv) []string {
						return Map(OfSlice(re.Names), func(string) string {
							return re.Interface.String()
						}).ToSlice()
					}).ToSlice()
				size := len(types)
				if size == 0 || size > 2 {
					panic("the return value must provide >= 1 & <= 2")
				}
				if types[0] == "error" {
					panic("the return [1] value must provide object")
				}
				if size == 2 && types[1] != "error" {
					panic("the return [1] value must provide error")
				}
			}

			var buf strings.Builder
			importPath := returns[0].ImportPath
			if importPath == "" {
				importPath = convert.ImportPath()
				// convert.Alias(alias)
			}

			inject := convert.tag.(annotations.Inject)
			iocClass := Or(returns[0].IsPointer && inject.N == "", "*", "") + Or(inject.N == "", importPath+"."+returns[0].Interface.Ext(), inject.N)
			if returns[0].Interface.Alias() != "" {
				if returns[0].Interface == "sdk.Container" {
					goto returnLabel
				}
				ip, ok := lookup.FindImportByAlias(returns[0].Interface.Alias())
				if ok {
					importPath = ip
					if ip != "github.com/iocgo/sdk" {
						imports, _ = Import(imports, returns[0].Interface.Alias(), importPath)
					}
				}
			}
		returnLabel:

			results, padding := joinReturn(returns)
			if !inject.IsLazy {
				activated = append(activated, fmt.Sprintf("container.AddInitialized(func() (err error) {\n\t_, err = sdk.InvokeBean[%s](container, \"%s\")\n\treturn })", returns[0].String(), iocClass))
			}

			pos := 1
			// 组件分配别名
			if n := inject.Alias; n != "" {
				buf.WriteString(fmt.Sprintf("container.Alias(\"%s\", \"%s\")\n", n, iocClass))
			}
			buf.WriteString(fmt.Sprintf("sdk.%s(container, \"%s\", func() (%s) {\n", Or(inject.Singleton, "ProvideBean", "ProvideTransient"), iocClass, results))
			{
				// 参数生成
				var vars []string
				i := -1
				args := convert.ExtractArguments(node.Lookup(), convert.node)
				for _, argv := range args {
					for _, n := range argv.Names {
						i++
						if n == "" || n == "_" {
							n = "var" + strconv.Itoa(pos)
							pos++
						}

						vars = append(vars, n)
						importPath = argv.ImportPath
						if importPath == "" {
							importPath = convert.ImportPath()
							// argv.Alias(alias)
						}

						if argv.Interface.Alias() != "" {
							if argv.Interface == "sdk.Container" {
								goto argvLabel
							}
							ip, ok := lookup.FindImportByAlias(argv.Interface.Alias())
							if ok {
								importPath = ip
								if ip != "github.com/iocgo/sdk" {
									imports, _ = Import(imports, argv.Interface.Alias(), importPath)
								}
							}
						}
					argvLabel:

						iocClass = Or(argv.IsPointer, "*", "") + importPath + "." + argv.Interface.Ext()
						if iocClass == "*github.com/iocgo/sdk.Container" {
							if n != "container" {
								buf.WriteString(fmt.Sprintf("%s := container\n", n))
							}
							continue
						}
						if argv.Interface == "string" {
							buf.WriteString(fmt.Sprintf("	%s := `%s`\n", n, strings.TrimSpace(inject.Config)))
							continue
						}

						var err error
						// 别名匹配
						if qualifier := inject.Qualifier; qualifier != "" {
							values := strings.Split(qualifier, ",")
							for _, value := range values {
								value = strings.TrimSpace(value)
								idx := strings.Index(value, "]:")
								if value[0] != '[' || idx == -1 {
									break
								}

								nums := strings.Split(value[1:idx], "~")
								n1, n2 := -1, -1
								n1, err = strconv.Atoi(nums[0])
								if err != nil {
									panic("qualifier value parse to int err: " + err.Error())
								}
								if len(nums) > 1 {
									n2, err = strconv.Atoi(nums[1])
									if err != nil {
										panic("qualifier value parse to int err: " + err.Error())
									}
								} else {
									n2 = n1
								}

								ok := false
								for num := n1; num <= n2; num++ {
									if i == num {
										iocClass = value[idx+2:]
										ok = true
										break
									}
								}
								if !ok {
									break
								}
							}
						}

						buf.WriteString(fmt.Sprintf(`	%s, err := sdk.InvokeBean[%s](container, "%s")`, n, argv.String(), iocClass))
						buf.WriteString("\n")
						buf.WriteString(fmt.Sprintf("	if err != nil {\n		var zero %s\n		return zero, err\n	}", returns[0].String()))
						buf.WriteString("\n")
					}
				}
				results = strings.Join(vars, ", ")
			}

			var1 := ""
			str := strings.Join(FlatMap(OfSlice(returns), func(t Argv) []string {
				return Map(OfSlice(t.Names), func(n string) string {
					if n == "" || n == "_" {
						n = "var" + strconv.Itoa(pos)
						pos++
					}
					if var1 == "" {
						var1 = n
					}
					return n
				}).ToSlice()
			}).ToSlice(), ", ")

			buf.WriteString(fmt.Sprintf("	%s := %s(%s)\n", str, convert.GetAstName(), results))
			// 执行初始化方法
			if init := inject.Initialize; init != "" {
				buf.WriteString("	// Invoke initialize method\n")
				buf.WriteString(fmt.Sprintf("	%s.%s()\n", var1, init))
			}
			buf.WriteString(fmt.Sprintf("	return %s%s })", str, Or(padding, ", nil", "")))
			codes = append(codes, buf.String())
		}
	}

	if len(activated) > 0 {
		codes = append(codes, "\t// Initialized instance\n\t//")
		codes = append(codes, activated...)
	}

	instance, err := template.New("ioc").Parse(iocTemplate)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"package": pkg,
		"imports": imports,
		"codes":   codes,
	}

	if err = instance.Execute(&buf, data); err != nil {
		panic(err)
	}

	ops = make(map[string][]byte)
	_, importPath, err := commandAsImportPath(rootPath)
	if err != nil {
		panic(err)
	}

	ops[filepath.Join(tempDir, importPath, "container.gen.go")] = buf.Bytes()
	return
}

func joinReturn(returns []Argv) (str string, padding bool) {
	results := FlatMap(OfSlice(returns), func(re Argv) []string {
		return Map(OfSlice(re.Names), func(string) string {
			return re.String()
		}).ToSlice()
	}).ToSlice()

	if len(results) == 1 {
		results = append(results, "error")
		padding = true
	}

	str = strings.Join(results, ", ")
	return
}

func Import(imports []Imported, alias, importPath string) ([]Imported, string) {
	pos := 1
	has := false
	change := false
	for _, ip := range imports {
		if ip.ImportPath == importPath {
			has = true
			break
		}

		if alias == "_" {
			continue
		}

		if ip.Alias == alias {
			change = true
			alias += strconv.Itoa(pos)
			pos++
		}
	}

	if alias == "" {
		idx := strings.LastIndex(importPath, "/")
		if idx > 0 {
			alias = importPath[idx+1:]
			change = true
		}
	}

	if has {
		return imports, alias
	}

	return append(imports, Imported{Or(change, alias, ""), importPath}), alias
}

func Or[T any](expr bool, a1 T, a2 T) T {
	if expr {
		return a1
	} else {
		return a2
	}
}
