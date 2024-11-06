package gen

import (
	annotation "github.com/bincooo/go-annotation/pkg"
	gen "github.com/iocgo/sdk/gen/annotation"
	"github.com/iocgo/sdk/gen/internal/core"
)

func Alias[T gen.M]() {
	core.Alias[T]()
}

func Process(root, tempDir, logLv string) {
	core.Root(root, tempDir)
	annotation.Process(root, logLv)
}
