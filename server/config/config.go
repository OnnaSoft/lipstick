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

type ProxyConfig struct {
	Address string `yaml:"address"`
}

type ManagerConfig struct {
	Address string `yaml:"address"`
}

type AdminConfig struct {
	Address string `yaml:"address"`
}

type TLSConfig struct {
	CertificatePath string `yaml:"certificate_path"`
	KeyPath         string `yaml:"key_path"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"ssl_mode"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Database int    `yaml:"database"`
}

type AppConfig struct {
	AdminSecretKey string         `yaml:"admin_secret_key"`
	Proxy          ProxyConfig    `yaml:"proxy"`
	Manager        ManagerConfig  `yaml:"manager"`
	Admin          AdminConfig    `yaml:"admin"`
	TLS            TLSConfig      `yaml:"tls"`
	Database       DatabaseConfig `yaml:"database"`
	Redis          RedisConfig    `yaml:"redis"`
}

var appConfig AppConfig

func loadConfig() {
	var configPath string
	var adminAddress string
	var managerAddress string
	var proxyAddress string
	var adminSecretKey string
	var tlsCert string
	var tlsKey string
	var redisHost string
	var redisPort int
	var redisPassword string
	var redisDatabase int

	// Default configuration
	defaultConfig := AppConfig{
		Admin: AdminConfig{
			Address: ":5052",
		},
		Manager: ManagerConfig{
			Address: ":5051",
		},
		Proxy: ProxyConfig{
			Address: ":5050",
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Database: 0,
		},
	}

	// Command-line flags
	flag.StringVar(&configPath, "c", "/etc/lipstick/config.yml", "Path to the configuration file")
	flag.StringVar(&adminAddress, "admin-addr", "", "Address for the admin API")
	flag.StringVar(&managerAddress, "manager-addr", "", "Address for WebSocket manager connections")
	flag.StringVar(&proxyAddress, "proxy-addr", "", "Address for the proxy")
	flag.StringVar(&adminSecretKey, "admin-secret", "", "Secret key for admin API authorization")
	flag.StringVar(&tlsCert, "tls-cert", "", "Path to the TLS certificate")
	flag.StringVar(&tlsKey, "tls-key", "", "Path to the TLS key")
	flag.StringVar(&redisHost, "redis-host", "", "Redis host")
	flag.IntVar(&redisPort, "redis-port", 0, "Redis port")
	flag.StringVar(&redisPassword, "redis-password", "", "Redis password")
	flag.IntVar(&redisDatabase, "redis-db", 0, "Redis database index")

	flag.Parse()

	// Read configuration from file
	f, err := os.Open(configPath)
	if err == nil {
		defer f.Close()
		buff, err := io.ReadAll(f)
		if err == nil {
			err = yaml.Unmarshal(buff, &defaultConfig)
			if err != nil {
				log.Printf("Error parsing configuration file: %v", err)
			}
		}
	}

	// Override with command-line arguments
	defaultConfig.Proxy.Address = helper.SetValue(proxyAddress, defaultConfig.Proxy.Address).(string)
	defaultConfig.Manager.Address = helper.SetValue(managerAddress, defaultConfig.Manager.Address).(string)
	defaultConfig.Admin.Address = helper.SetValue(adminAddress, defaultConfig.Admin.Address).(string)
	defaultConfig.AdminSecretKey = helper.SetValue(adminSecretKey, defaultConfig.AdminSecretKey).(string)
	defaultConfig.TLS.CertificatePath = helper.SetValue(tlsCert, defaultConfig.TLS.CertificatePath).(string)
	defaultConfig.TLS.KeyPath = helper.SetValue(tlsKey, defaultConfig.TLS.KeyPath).(string)

	// Redis configuration
	defaultConfig.Redis.Host = helper.SetValue(redisHost, defaultConfig.Redis.Host).(string)
	defaultConfig.Redis.Port = helper.SetValue(redisPort, defaultConfig.Redis.Port).(int)
	defaultConfig.Redis.Password = helper.SetValue(redisPassword, defaultConfig.Redis.Password).(string)
	defaultConfig.Redis.Database = helper.SetValue(redisDatabase, defaultConfig.Redis.Database).(int)

	appConfig = defaultConfig
}

func GetConfig() (AppConfig, error) {
	if appConfig.AdminSecretKey != "" {
		return appConfig, nil
	}

	loadConfig()

	if appConfig.AdminSecretKey == "" {
		log.Fatal(errors.New("failed to load configuration"))
	}

	return appConfig, nil
}
