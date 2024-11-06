// 该包下仅提供给iocgo工具使用的，不需要理会 Injects 的错误，在编译过程中生成
package scan

import (
	"github.com/iocgo/sdk"
	env "github.com/iocgo/sdk/env"
)

func Injects(container *sdk.Container) error {
	return env.Injects(container)
}
