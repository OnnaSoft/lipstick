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
	var redisHost, redisPassword string
	var redisPort, redisDatabase, redisPoolSize, redisMinIdleConns, redisPoolTimeout int
	var dbHost, dbUser, dbPassword, dbName, dbSSLMode string
	var dbPort int

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
	flag.IntVar(&redisPoolSize, "redis-pool-size", 0, "Redis pool size")
	flag.IntVar(&redisMinIdleConns, "redis-min-idle-conns", 0, "Redis minimum idle connections")
	flag.IntVar(&redisPoolTimeout, "redis-pool-timeout", 0, "Redis pool timeout in seconds")
	flag.StringVar(&dbHost, "db-host", "", "Database host")
	flag.IntVar(&dbPort, "db-port", 0, "Database port")
	flag.StringVar(&dbUser, "db-user", "", "Database user")
	flag.StringVar(&dbPassword, "db-password", "", "Database password")
	flag.StringVar(&dbName, "db-name", "", "Database name")
	flag.StringVar(&dbSSLMode, "db-ssl-mode", "", "Database SSL mode")

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

	if defaultConfig.AdminSecretKey != "" {
		appConfig = defaultConfig
		return
	}

	// Override with environment variables if no CLI or file value is provided
	adminAddress = setValue(adminAddress, os.Getenv("ADMIN_ADDR")).(string)
	managerAddress = setValue(managerAddress, os.Getenv("MANAGER_ADDR")).(string)
	proxyAddress = setValue(proxyAddress, os.Getenv("PROXY_ADDR")).(string)
	adminSecretKey = setValue(adminSecretKey, os.Getenv("ADMIN_SECRET_KEY")).(string)
	tlsCert = setValue(tlsCert, os.Getenv("TLS_CERT")).(string)
	tlsKey = setValue(tlsKey, os.Getenv("TLS_KEY")).(string)
	redisHost = setValue(redisHost, os.Getenv("REDIS_HOST")).(string)
	redisPort = setValue(redisPort, parseEnvInt("REDIS_PORT", redisPort)).(int)
	redisPassword = setValue(redisPassword, os.Getenv("REDIS_PASSWORD")).(string)
	redisDatabase = setValue(redisDatabase, parseEnvInt("REDIS_DB", redisDatabase)).(int)
	redisPoolSize = setValue(redisPoolSize, parseEnvInt("REDIS_POOL_SIZE", redisPoolSize)).(int)
	redisMinIdleConns = setValue(redisMinIdleConns, parseEnvInt("REDIS_MIN_IDLE_CONNS", redisMinIdleConns)).(int)
	redisPoolTimeout = setValue(redisPoolTimeout, parseEnvInt("REDIS_POOL_TIMEOUT", redisPoolTimeout)).(int)
	dbHost = setValue(dbHost, os.Getenv("DB_HOST")).(string)
	dbPort = setValue(dbPort, parseEnvInt("DB_PORT", dbPort)).(int)
	dbUser = setValue(dbUser, os.Getenv("DB_USER")).(string)
	dbPassword = setValue(dbPassword, os.Getenv("DB_PASSWORD")).(string)
	dbName = setValue(dbName, os.Getenv("DB_NAME")).(string)
	dbSSLMode = setValue(dbSSLMode, os.Getenv("DB_SSL_MODE")).(string)

	// Apply values
	defaultConfig.Admin.Address = adminAddress
	defaultConfig.Manager.Address = managerAddress
	defaultConfig.Proxy.Address = proxyAddress
	defaultConfig.AdminSecretKey = adminSecretKey
	defaultConfig.TLS.CertificatePath = tlsCert
	defaultConfig.TLS.KeyPath = tlsKey
	defaultConfig.Redis.Host = redisHost
	defaultConfig.Redis.Port = redisPort
	defaultConfig.Redis.Password = redisPassword
	defaultConfig.Redis.Database = redisDatabase
	defaultConfig.Redis.PoolSize = redisPoolSize
	defaultConfig.Redis.MinIdleConns = redisMinIdleConns
	defaultConfig.Redis.PoolTimeout = redisPoolTimeout
	defaultConfig.Database.Host = dbHost
	defaultConfig.Database.Port = dbPort
	defaultConfig.Database.User = dbUser
	defaultConfig.Database.Password = dbPassword
	defaultConfig.Database.Database = dbName
	defaultConfig.Database.SSLMode = dbSSLMode

	appConfig = defaultConfig
}

// Helper function to parse environment variables as integers
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
