package db

import (
	"fmt"
	"log"
	"os"

	"github.com/OnnaSoft/lipstick/server/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var defaultConnection *gorm.DB

func NewConnection(conf config.DatabaseConfig) (*gorm.DB, error) {
	// Construir el DSN para PostgreSQL
	dsn := fmt.Sprintf(
		"host=%s user=%s dbname=%s sslmode=%s password=%s",
		conf.Host, conf.User, conf.Database, conf.SSLMode, conf.Password,
	)

	// Abrir conexión con PostgreSQL (se fuerza el uso de postgres)
	connection, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Activar el modo de depuración si está habilitado
	if os.Getenv("DEBUG") == "true" {
		db, _ := connection.DB()
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		fmt.Println("DEBUG mode: Database logging enabled.")
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
		sqlDB, err := defaultConnection.DB()
		if err != nil {
			log.Printf("failed to get native DB connection: %v", err)
		}
		sqlDB.Close()
	}
}

func Migrate(conf config.DatabaseConfig) {
	connection, err := GetConnection(conf)
	if err != nil {
		log.Fatal(err)
	}

	if err := connection.AutoMigrate(&Domain{}, &DailyConsumption{}); err != nil {
		log.Fatal(err.Error())
	}
}
