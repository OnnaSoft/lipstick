package config

import (
	"crypto/tls"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"strconv"

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

func (t *TLSConfig) GetTLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair(t.CertificatePath, t.KeyPath)
	if err != nil {
		log.Printf("Error loading TLS certificate: %v", err)
		return nil
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
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
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Password     string `yaml:"password"`
	Database     int    `yaml:"database"`
	PoolSize     int    `yaml:"pool_size"`
	MinIdleConns int    `yaml:"min_idle_conns"`
	PoolTimeout  int    `yaml:"pool_timeout"`
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
	var adminAddress, managerAddress, proxyAddress, adminSecretKey, tlsCert, tlsKey string
	var redisHost, redisPassword, rabbitMQHost, rabbitMQUser, rabbitMQPassword, rabbitMQVHost string
	var redisPort, redisDatabase, redisPoolSize, redisMinIdleConns, redisPoolTimeout, rabbitMQPort int
	var dbHost, dbUser, dbPassword, dbName, dbSSLMode string
	var dbPort int

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
			Host:         "localhost",
			Port:         6379,
			Database:     0,
			PoolSize:     10,
			MinIdleConns: 3,
			PoolTimeout:  30,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "",
			Database: "app_db",
			SSLMode:  "disable",
		},
	}

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
	flag.IntVar(&redisPoolSize, "redis-pool-size", 0, "Redis pool size")
	flag.IntVar(&redisMinIdleConns, "redis-min-idle-conns", 0, "Redis minimum idle connections")
	flag.IntVar(&redisPoolTimeout, "redis-pool-timeout", 0, "Redis pool timeout in seconds")
	flag.StringVar(&rabbitMQHost, "rabbitmq-host", "", "RabbitMQ host")
	flag.IntVar(&rabbitMQPort, "rabbitmq-port", 0, "RabbitMQ port")
	flag.StringVar(&rabbitMQUser, "rabbitmq-user", "", "RabbitMQ user")
	flag.StringVar(&rabbitMQPassword, "rabbitmq-password", "", "RabbitMQ password")
	flag.StringVar(&rabbitMQVHost, "rabbitmq-vhost", "", "RabbitMQ virtual host")
	flag.StringVar(&dbHost, "db-host", "", "Database host")
	flag.IntVar(&dbPort, "db-port", 0, "Database port")
	flag.StringVar(&dbUser, "db-user", "", "Database user")
	flag.StringVar(&dbPassword, "db-password", "", "Database password")
	flag.StringVar(&dbName, "db-name", "", "Database name")
	flag.StringVar(&dbSSLMode, "db-ssl-mode", "", "Database SSL mode")

	flag.Parse()

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

	appConfig = defaultConfig
}

func parseEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
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
