package config

import (
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Proxy struct {
		Addr string `yaml:"addr"`
	} `yaml:"proxy"`
	Manager struct {
		Addr string `yaml:"addr"`
	} `yaml:"manager"`
}

var config interface{}
var configPath string = ""

func getConfigArgs() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	configPathDefault := path.Join(dir, "config.client.yml")
	flag.StringVar(&configPath, "c", configPathDefault, "config path")

	managerAddr := ""
	managerAddrDefault := path.Join(dir, ":8080")
	flag.StringVar(&managerAddr, "m", managerAddrDefault, "Manager address")

	proxyAddr := ""
	proxyAddrDefault := path.Join(dir, ":8081")
	flag.StringVar(&proxyAddr, "p", proxyAddrDefault, "Proxy address")

	flag.Parse()
}

func GetConfig() (Config, error) {
	if config != nil {
		return config.(Config), nil
	}

	if configPath == "" {
		getConfigArgs()
	}
	result := Config{}

	f, err := os.Open(configPath)
	if err != nil {
		return result, err
	}
	buff, err := io.ReadAll(f)
	if err != nil {
		return result, err
	}
	err = yaml.Unmarshal(buff, &result)
	if err != nil {
		return result, err
	}
	config = result

	return config.(Config), nil
}
