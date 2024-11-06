package exec

import (
	"github.com/iocgo/sdk/gen/internal/logger"
	"os"
	"path/filepath"
	"strings"
)

func link(args []string) {
	var cfg string
	buildmode := false
	for _, arg := range args {
		if arg == "-buildmode=exe" ||
			// windows
			arg == "-buildmode=pie" {
			buildmode = true
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if strings.Contains(arg, filepath.Join("b001", "importcfg.link")) {
			cfg = arg
		}
	}
	logger.Debug("cfg", cfg)
	if !buildmode || cfg == "" {
		return
	}
	if cmdFlag.ClearWork {
		exitDo = func() {
			_ = os.RemoveAll(tempDir)
		}
	}
}
