package exec

import (
	"fmt"
	"github.com/iocgo/sdk/gen"
	"github.com/iocgo/sdk/gen/internal/logger"
	"github.com/iocgo/sdk/gen/internal/meta"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var (
	packageIn        *meta.PackageInfo
	goRootPackageDir = os.Getenv("GOROOT") + "/src/"
)

func compile(args []string) (nArgs []string, err error) {
	var files []string

	packageName := ""
	packageDir := ""
	for i, arg := range args {
		if arg == "-p" && i+1 < len(args) {
			packageName = args[i+1]
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}

		if strings.HasSuffix(arg, ".go") {
			packageDir = filepath.Dir(arg)
			files = slices.Clone(args[i:])
			nArgs = slices.Clone(args[:i])
			break
		}
	}

	logger.Debug("source files: ", files)
	if strings.HasPrefix(packageDir, goRootPackageDir) || strings.Contains(packageDir, "/golang.org/") {
		return args, nil
	}

	{
		packageDir, _ = filepath.Abs(packageDir)
		packageIn, err = meta.GetPackageInfo(packageDir)
		if err != nil || packageIn.Module.Path == "" {
			logger.Warn(fmt.Sprintf("doesn't seem to be a Go project [1] '%s' :", packageDir), err)
			return args, nil
		}
	}

	projectName := packageIn.Module.Path
	if (packageName != "main" && !strings.HasPrefix(packageName, projectName)) || len(files) == 0 {
		logger.Warn(fmt.Sprintf("doesn't seem to be a Go project [2] '%s' :", packageName), err)
		return args, nil
	}

	logLv := "e"
	switch cmdFlag.Level {
	case "all", "debug", "info", "warn":
		logLv = "w"
	case "error":
		logLv = "e"
	case "close":
		logLv = "f"
	}
	// 中间代码生成
	gen.Process(packageDir, tempDir, logLv)

	joinPath := filepath.Join(tempDir, packageIn.ImportPath)
	if !meta.IsExist(joinPath) {
		goto label
	}

	err = filepath.Walk(joinPath, func(path string, info os.FileInfo, err error) (_ error) {
		if err != nil {
			return err
		}

		if info.IsDir() ||
			filepath.Dir(path) != joinPath ||
			filepath.Ext(path) != ".go" {
			return
		}
		logger.Debug("each tempDir children file: ", path)

		// 同名覆盖
		for i := range files {
			if filepath.Base(files[i]) == filepath.Base(path) {
				logger.Debug("rewrite file", files[i], "=>", path)
				files = append(files[:i], files[i+1:]...)
				nArgs = append(nArgs, path)
				return
			}
		}

		// 中间代码文件附加
		nArgs = append(nArgs, path)
		return
	})

	if err != nil {
		return
	}

	logger.Debug("confirm new Args : ", append(nArgs, files...))

label:
	nArgs = append(nArgs, files...)
	return
}
