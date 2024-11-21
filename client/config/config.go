package config

import (
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/juliotorresmoreno/lipstick/helper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	APISecret string   `yaml:"api_secret"` // API secret for authentication
	ServerURL string   `yaml:"server_url"` // URL of the server manager
	ProxyPass []string `yaml:"proxy_pass"` // List of proxy targets
	Workers   int      `yaml:"workers"`    // Number of worker routines
}

var config *Config

// loadConfig reads the configuration file and merges it with CLI arguments
func loadConfig() {
	var (
		configPath string
		serverURL  string
		proxyPass  string
		apiSecret  string
		workers    int
	)

	// Default configuration
	result := Config{}

	// Set default configuration path relative to the executable's location
	execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Printf("Error getting executable directory: %v", err)
	}
	defaultConfigPath := filepath.Join(execDir, "config.client.yml")

	// CLI Flags
	flag.StringVar(&configPath, "c", defaultConfigPath, "Path to the configuration file")
	flag.StringVar(&serverURL, "s", "http://localhost:5051", "URL for the server manager WebSocket")
	flag.StringVar(&proxyPass, "p", "tcp://127.0.0.1:12000", "Proxy targets separated by spaces")
	flag.StringVar(&apiSecret, "k", "", "API secret for authenticating nodes")
	flag.Parse()

	// Load YAML config file
	if file, err := os.Open(configPath); err == nil {
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			log.Printf("Error reading config file: %v", err)
		} else if err := yaml.Unmarshal(content, &result); err != nil {
			log.Printf("Error parsing config file: %v", err)
		}
	}

	// Merge CLI arguments
	result.ServerURL = helper.SetValue(serverURL, result.ServerURL).(string)
	if proxyPass != "" {
		result.ProxyPass = strings.Fields(proxyPass)
	}
	result.APISecret = helper.SetValue(apiSecret, result.APISecret).(string)
	result.Workers = helper.SetValue(workers, result.Workers).(int)

	// Store in global config
	config = &result
}

// GetConfig provides the application configuration
func GetConfig() (*Config, error) {
	if config == nil {
		loadConfig()
		if config == nil {
			return nil, errors.New("failed to load configuration")
		}
	}
	return config, nil
}
