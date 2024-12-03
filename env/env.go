package env

import (
	"bytes"
	"crypto/tls"
	_ "github.com/iocgo/sdk"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	Env *Environment
)

type Environment struct {
	*viper.Viper
	Args []string

	Env  []string
	path string
}

// @Inject(lazy = "true")
func New() (env *Environment, err error) {
	path := "config.yaml"
	environ := os.Environ()

	if argsLen := len(os.Args); argsLen > 0 && strings.HasSuffix(os.Args[argsLen-1], ".yaml") {
		path = os.Args[argsLen-1]
		goto label
	}

	for _, item := range environ {
		if strings.HasPrefix(item, "CONFIG_PATH=") && len(item) > 12 {
			path = item[12:]
			break
		}
	}

label:
	config, err := readConfig(path)
	if err != nil {
		return
	}

	vip := viper.New()
	vip.SetConfigType("yaml")
	if err = vip.ReadConfig(bytes.NewReader(config)); err != nil {
		return
	}

	env = &Environment{
		path:  path,
		Env:   os.Environ(),
		Args:  os.Args[1:],
		Viper: vip,
	}
	Env = env
	return
}

func readConfig(path string) (config []byte, err error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		var response *http.Response
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
		response, err = client.Get(path)
		if err != nil {
			return
		}
		defer response.Body.Close()
		config, err = io.ReadAll(response.Body)
		return
	}

	config, err = os.ReadFile(path)
	return
}
