package exec

import (
	"flag"
	"fmt"
	"github.com/iocgo/sdk/gen/internal/logger"
	"log"
	"os"
	"strings"
)

const version = `v1.0.1 beta`
const opensourceUrl = `https://github.com/iocgo/sdk`

type CmdFlag struct {
	Level     string // -d.log
	TempDir   string // -d.tempDir
	ClearWork bool   // -d.clearWork
	Version   string // -version

	// go build args
	toolPath  string
	chainName string
	chainArgs []string
}

func initUseFlag() {
	flag.StringVar(&cmdFlag.Level,
		"d.log",
		"error",
		"output log level. all/debug/info/warn/error/close")
	flag.StringVar(&cmdFlag.TempDir,
		"d.tempDir",
		"",
		"tool workspace dir. default same as go build workspace")
	flag.BoolVar(&cmdFlag.ClearWork,
		"d.clearWork",
		true,
		"empty workspace when compilation is complete")
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		_, _ = fmt.Fprintf(flag.CommandLine.Output(),
			"gen [-d.log] [-d.tempDir] chainToolPath chainArgs\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	switch cmdFlag.Level {
	case "all":
		logger.Log.Level = logger.LevelAll
	case "debug":
		logger.Log.Level = logger.LevelDebug
	case "info":
		logger.Log.Level = logger.LevelInfo
	case "warn":
		logger.Log.Level = logger.LevelWarn
	case "error", "":
		logger.Log.Level = logger.LevelError
	case "close":
		logger.Log.Level = logger.LevelClose
	}
	log.SetPrefix("gen : ")
	if logger.Log.Level < logger.LevelDebug {
		log.SetFlags(0)
	}
	if cmdFlag.TempDir != "" {
		tempDir = cmdFlag.TempDir
	}
	cmdFlag.toolPath = os.Args[0]
	goToolDir := os.Getenv("GOTOOLDIR")
	if goToolDir == "" {
		logger.Info("env key `GOTOOLDIR` not found")
	}
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(),
			"gen %s , %s\n", version, opensourceUrl)
		os.Exit(0)
	}
	for i, arg := range os.Args[1:] {
		if goToolDir != "" && strings.HasPrefix(arg, goToolDir) {
			cmdFlag.chainName = arg
			if len(os.Args[1:]) > i+1 {
				cmdFlag.chainArgs = os.Args[i+2:]
			}
			break
		}
	}
}

var cmdFlag = &CmdFlag{}
