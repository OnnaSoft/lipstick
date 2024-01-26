package config

import (
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/juliotorresmoreno/lipstick/helper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Keyword   string   `yaml:"keyword"`
	ServerUrl string   `yaml:"serverUrl"`
	ProxyPass []string `yaml:"proxyPass"`
}

var config interface{}

func loadConfig() {
	var configPath string = ""
	var serverUrl string = ""
	var proxyPass string = ""
	var secret = ""

	result := Config{}
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	configPathDefault := path.Join(dir, "config.client.yml")
	flag.StringVar(&configPath, "c", configPathDefault, "config path")

	flag.StringVar(&serverUrl, "s", "ws://localhost:5051/ws", "Where you are listening your server manager port")
	flag.StringVar(&proxyPass, "p", "127.0.0.1:12000", "Host/port where you want connect from the remote server")
	flag.StringVar(&secret, "k", "", "Private secret use to autenticate nodes")

	flag.Parse()

	f, err := os.Open(configPath)
	if err == nil {
		buff, err := io.ReadAll(f)
		if err != nil {
			return
		}
		err = yaml.Unmarshal(buff, &result)
		if err != nil {
			return
		}
	}

	result.ServerUrl = helper.SetValue(serverUrl, result.ServerUrl).(string)
	proxies := strings.Split(proxyPass, " ")
	if len(proxies) > 0 {
		result.ProxyPass = proxies
	}
	result.Keyword = helper.SetValue(secret, result.Keyword).(string)

	config = result
}

func GetConfig() (Config, error) {
	if config != nil {
		return config.(Config), nil
	}

	loadConfig()

	if config == nil {
		log.Fatal(errors.New("could not load config"))
	}

	return config.(Config), nil
}
