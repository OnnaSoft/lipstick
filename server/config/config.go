package config

import (
	"errors"
	"flag"
	"io"
	"log"
	"os"

	"github.com/juliotorresmoreno/lipstick/helper"
	"gopkg.in/yaml.v3"
)

type Proxy struct {
	Addr string `yaml:"addr"`
}

type Manager struct {
	Addr string `yaml:"addr"`
}

type Admin struct {
	Addr string `yaml:"addr"`
}

type Certs struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type Database struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DbName   string `yaml:"dbname"`
	SslMode  string `yaml:"sslmode"`
}

type Config struct {
	Keyword  string   `yaml:"keyword"`
	Proxy    Proxy    `yaml:"proxy"`
	Manager  Manager  `yaml:"manager"`
	Admin    Admin    `yaml:"admin"`
	Certs    Certs    `yaml:"certs"`
	Database Database `yaml:"database"`
}

var config interface{}

func loadConfig() {
	var configPath = ""
	var adminAddr = ""
	var managerAddr = ""
	var proxyAddr = ""
	var secret = ""
	var cert = ""
	var key = ""

	result := Config{
		Admin: Admin{
			Addr: ":5052",
		},
		Manager: Manager{
			Addr: ":5051",
		},
		Proxy: Proxy{
			Addr: ":5050",
		},
	}

	configPathDefault := "/etc/lipstick/config.yml"
	flag.StringVar(&configPath, "c", configPathDefault, "config path")
	flag.StringVar(&adminAddr, "a", "", "Port where you will get all requests from local network or internet")
	flag.StringVar(&managerAddr, "m", "", "Port where your client will connect via websocket. You can manage it in your firewall")
	flag.StringVar(&proxyAddr, "p", "", "Port where you will get all requests from local network or internet")
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
