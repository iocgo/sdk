package meta

import (
	"encoding/json"
	"fmt"
	"github.com/iocgo/sdk/stream"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	projectDir, _      = os.Getwd()
	goRootSrcDir       = os.Getenv("GOROOT") + "/src/"
	goPackageModDir    = os.Getenv("GOPATH") + "/pkg/mod/"
	goPackageVendorDir = projectDir + "/vendor/"

	baseTypes = []string{
		"uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64",

		"float32", "float64", "complex64", "complex128",

		"byte", "uint", "int",

		"map",

		"bool",

		"rune",
		"string",
		"uintptr",
		"interface",
		"error",
	}
)

type PackageInfo struct {
	Dir,
	ImportPath,
	Name,
	Target,
	Root,
	StaleReason string
	Stale  bool
	Module struct {
		Main bool
		Path,
		Dir,
		GoMod,
		GoVersion string
	}
	Match,
	GoFiles,
	Imports, // TODO remove -find
	Deps []string
}

func (in PackageInfo) FindPackageDirFor(packageName string) (string, error) {
	if strings.HasPrefix(packageName, in.Module.Path) {
		return filepath.Join(in.Module.Dir, strings.TrimPrefix(packageName, in.Module.Path)), nil
	}

	for _, str := range in.Deps {
		// 命中该根包名
		root := strings.Split(str, "@")[0]
		if !strings.HasPrefix(packageName, root) {
			continue
		}

		var packageDir string
		if root == packageName {
			// 根包名匹配
			packageDir = str
		} else {
			// 子包名匹配
			if idx := strings.Index(str, "=>@"); idx > 0 {
				str = str[idx+3:]
			}
			packageDir = str + strings.TrimPrefix(packageName, root)
		}

		if strings.HasPrefix(packageDir, "./") || strings.HasPrefix(packageDir, "../") {
			if IsExist(packageDir) {
				return packageDir, nil
			}
			continue
		}

		// vendor 目录查找
		if dir := filepath.Join(goPackageVendorDir, packageDir); IsExist(dir) {
			return dir, nil
		}

		// gopath 目录查找
		if dir := filepath.Join(goPackageModDir, packageDir); IsExist(dir) {
			return dir, nil
		}
	}

	// 主项目查找

	// goroot 目录查找
	if dir := filepath.Join(goRootSrcDir, packageName); IsExist(dir) {
		return dir, nil
	}

	return "", fmt.Errorf("package %s not found", packageName)
}

func GetInterfaceInfo(dirOrFile, name string) (spec *ast.TypeSpec, err error) {
	tokenSet := token.NewFileSet()
	var fs []*ast.File

	if strings.HasSuffix(dirOrFile, ".go") {
		var f *ast.File
		f, err = parser.ParseFile(tokenSet, dirOrFile, nil, parser.ParseComments)
		if err != nil {
			return
		}
		fs = append(fs, f)
	} else {
		err = filepath.Walk(dirOrFile, func(path string, info os.FileInfo, err error) (_ error) {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(path, ".go") || filepath.Dir(path) != dirOrFile {
				return
			}

			var f *ast.File
			f, err = parser.ParseFile(tokenSet, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}

			fs = append(fs, f)
			return
		})

		if err != nil {
			return
		}
	}

	if idx := strings.LastIndex(name, "."); idx > 0 {
		name = name[idx+1:]
	}

	var imports []*ast.ImportSpec

	for _, f := range fs {
		ast.Inspect(f, func(node ast.Node) (ok bool) {
			switch expr := node.(type) {
			case *ast.TypeSpec:
				if expr.Name.Name == name {
					spec = expr
					imports = f.Imports
					return
				}
			}
			return true
		})
	}

	if spec == nil {
		err = fmt.Errorf("could not find interface info for %s", name)
	}

	err = eachMethod(imports, spec.Type.(*ast.InterfaceType).Methods)
	return
}

func FindPackageByImports(imports []*ast.ImportSpec, alias string) (*PackageInfo, error) {
	if len(imports) == 0 {
		return nil, fmt.Errorf("no packages found for `%s`", alias)
	}

	packageInfo, err := GetPackageInfo("")
	if err != nil {
		return nil, err
	}

	for _, imp := range imports {
		if imp.Name != nil && imp.Name.Name == alias {
			dir, err := packageInfo.FindPackageDirFor(imp.Path.Value)
			if err != nil {
				return nil, err
			}
			return GetPackageInfo(dir)
		}
	}

	for _, imp := range imports {
		dir, err := packageInfo.FindPackageDirFor(imp.Path.Value[1 : len(imp.Path.Value)-1])
		if err != nil {
			return nil, err
		}
		packageIn, err := GetPackageInfo(dir)
		if err != nil {
			return nil, err
		}
		if packageIn.Name == alias {
			return packageIn, nil
		}
	}

	return nil, fmt.Errorf("package `%s` not found", alias)
}

func GetPackageInfo(pkgPath string) (*PackageInfo, error) {
	if pkgPath == "" {
		pkgPath = projectDir
	}

	command := []string{"go", "list", "-json", "-find"}
	if pkgPath != "" && pkgPath != "main" {
		command = append(command, pkgPath)
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = projectDir
	cmd.Env = os.Environ()
	bf, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	p := &PackageInfo{}
	err = json.Unmarshal(bf, p)
	if err != nil {
		return nil, err
	}

	_projectDir := projectDir
	command = []string{"go", "list", "-m", "all"}
	if pkgPath != "" && pkgPath != "main" {
		_projectDir = pkgPath
	}
	cmd = exec.Command(command[0], command[1:]...)
	cmd.Dir = _projectDir
	cmd.Env = os.Environ()
	bf, err = cmd.Output()
	if err != nil {
		return p, nil
	}
	slice := strings.Split(string(bf), "\n")
	p.Deps = append(p.Deps, stream.Map(stream.OfSlice(slice), func(str string) string {
		return strings.Join(strings.Split(str, " "), "@")
	}).ToSlice()...)

	return p, nil
}

func IsExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func IsBaseTyp(name string) bool {
	for _, t := range baseTypes {
		if name == t {
			return true
		}

		// TODO more ...
	}
	return false
}

func eachMethod(imports []*ast.ImportSpec, methods *ast.FieldList) error {
	for _, method := range methods.List {
		switch expr := method.Type.(type) {
		case *ast.SelectorExpr:
			alias := expr.X.(*ast.Ident).Name
			packageInfo, err := FindPackageByImports(imports, alias)
			if err != nil {
				return err
			}
			typeSpec, err := GetInterfaceInfo(packageInfo.Dir, expr.Sel.Name)
			if err != nil {
				return err
			}
			method.Type = typeSpec.Type
			if err = eachMethod(imports, method.Type.(*ast.InterfaceType).Methods); err != nil {
				return err
			}
		}
	}
	return nil
}
