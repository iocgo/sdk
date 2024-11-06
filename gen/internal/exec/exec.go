package exec

import (
	"github.com/iocgo/sdk/gen/internal/logger"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var (
	tempDir = path.Join(os.TempDir(), "go_build_gen_works")
	exitDo  = func() {}
)

func Process() {
	initUseFlag()
	initTempDir()

	if cmdFlag.chainName == "" {
		logger.Error("currently not in a compilation chain environment and cannot be used")
	}

	chainName := cmdFlag.chainName
	chainArgs := cmdFlag.chainArgs
	toolName := filepath.Base(chainName)

	var err error
	switch strings.TrimSuffix(toolName, ".exe") {
	case "compile":
		chainArgs, err = compile(chainArgs)
	case "link":
		link(chainArgs)
		defer func() {
			logger.Debug("exitDo() begin")
			exitDo()
			logger.Debug("exitDo() end")
		}()
	}

	if err != nil {
		logger.Error(err)
	}
	// build
	cmd := exec.Command(chainName, chainArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err = cmd.Run(); err != nil {
		// logger.Error("run toolchain err", chainName, err)
	}
}

func initTempDir() {
	if err := os.MkdirAll(tempDir, 0777); err != nil {
		logger.Error("Init() fail, os.MkdirAll tempDir", err)
	}
}
