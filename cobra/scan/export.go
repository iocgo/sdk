// 该包下仅提供给iocgo工具使用的，不需要理会 Injects 的错误，在编译过程中生成
package scan

import (
	"github.com/iocgo/sdk"
	"github.com/iocgo/sdk/cobra"
)

// cobra 自动装配，需要定义一个名为rootCobra的cobra.ICobra实例

func Injects(container *sdk.Container) (_ error) {
	sdk.ProvideBean[sdk.Initializer](container, "cobraInitializer", func() (i sdk.Initializer, err error) {
		i = CobraInitialized()
		return
	})
	return
}

func CobraInitialized() sdk.Initializer {
	return sdk.InitializedWrapper(1000, func(container *sdk.Container) (err error) {
		c, err := sdk.InvokeBean[cobra.ICobra](container, "rootCobra")
		if err != nil {
			return
		}
		return c.Command().Execute()
	})
}
