package config

import (
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/juliotorresmoreno/lipstick/helper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Keyword string `yaml:"keyword"`
	Proxy   struct {
		Addr string `yaml:"addr"`
	} `yaml:"proxy"`
	Manager struct {
		Addr string `yaml:"addr"`
	} `yaml:"manager"`
	Certs struct {
		Cert string `yaml:"cert"`
		Key  string `yaml:"key"`
	} `yaml:"certs"`
}

var config interface{}

func loadConfig() {
	var configPath = ""
	var managerAddr = ""
	var proxyAddr = ""
	var secret = ""
	var cert = ""
	var key = ""

	result := Config{}
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	configPathDefault := path.Join(dir, "config.client.yml")
	flag.StringVar(&configPath, "c", configPathDefault, "config path")
	flag.StringVar(&managerAddr, "m", ":5051", "Port where your client will connect via websocket. You can manage it in your firewall")
	flag.StringVar(&proxyAddr, "p", ":5050", "Port where you will get all requests from local network or internet")
	flag.StringVar(&secret, "k", "", "Private secret use to autenticate nodes")

	flag.StringVar(&cert, "cert", "", "Path to the certificate file")
	flag.StringVar(&key, "key", "", "Path to the key file")

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

	result.Proxy.Addr = helper.SetValue(proxyAddr, result.Proxy.Addr).(string)
	result.Manager.Addr = helper.SetValue(managerAddr, result.Manager.Addr).(string)
	result.Keyword = helper.SetValue(secret, result.Keyword).(string)

	result.Certs.Cert = helper.SetValue(cert, result.Certs.Cert).(string)
	result.Certs.Key = helper.SetValue(key, result.Certs.Key).(string)

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
