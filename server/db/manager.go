package db

import (
	"log"
	"os"

	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var defaultConnection *gorm.DB

func NewConnection(conf config.DatabaseConfig) (*gorm.DB, error) {
	dsn := "host=" + conf.Host +
		" user=" + conf.User +
		" dbname=" + conf.Database +
		" sslmode=" + conf.SSLMode +
		" password=" + conf.Password
	connection, err := gorm.Open("postgres", dsn)
	if os.Getenv("DEBUG") == "true" {
		connection.LogMode(true)
	}

	if err != nil {
		return nil, err
	}

	return connection, nil
}

func GetConnection(conf config.DatabaseConfig) (*gorm.DB, error) {
	if defaultConnection == nil {
		connection, err := NewConnection(conf)
		if err != nil {
			return nil, err
		}
		defaultConnection = connection
	}

	return defaultConnection, nil
}

func CloseConnection() {
	if defaultConnection != nil {
		defaultConnection.Close()
	}
}

func Migrate(conf config.DatabaseConfig) {
	connection, err := GetConnection(conf)
	if err != nil {
		log.Fatal(err)
	}

	if tx := connection.AutoMigrate(&Domain{}, &DailyConsumption{}); tx.Error != nil {
		log.Fatal(tx.Error)
	}
}
