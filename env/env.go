package env

import (
	"bytes"
	"github.com/spf13/viper"
	"os"

	_ "github.com/iocgo/sdk"
)

type Environment struct {
	Config *viper.Viper
	Args   []string

	Env  []string
	path string
}

// @Inject()
func New() (env *Environment, err error) {
	path := "config.yaml"
	config, err := os.ReadFile("config.yaml")
	if err != nil {
		return
	}

	vip := viper.New()
	vip.SetConfigType("yaml")
	if err = vip.ReadConfig(bytes.NewReader(config)); err != nil {
		return
	}

	env = &Environment{
		path:   path,
		Env:    os.Environ(),
		Args:   os.Args[1:],
		Config: vip,
	}
	return
}
