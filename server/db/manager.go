package db

import (
	"log"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/juliotorresmoreno/lipstick/server/config"
)

var defaultConnection *gorm.DB

func NewConnection() (*gorm.DB, error) {
	conf, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	dsn := "host=" + conf.Database.Host +
		" user=" + conf.Database.User +
		" dbname=" + conf.Database.DbName +
		" sslmode=" + conf.Database.SslMode +
		" password=" + conf.Database.Password
	connection, err := gorm.Open("postgres", dsn)
	if os.Getenv("DEBUG") == "true" {
		connection.LogMode(true)
	}

	if err != nil {
		return nil, err
	}

	return connection, nil
}

func GetConnection() (*gorm.DB, error) {
	if defaultConnection == nil {
		connection, err := NewConnection()
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

func Migrate() {
	connection, err := GetConnection()
	if err != nil {
		log.Fatal(err)
	}

	if tx := connection.AutoMigrate(&Domain{}); tx.Error != nil {
		log.Fatal(tx.Error)
	}
}
