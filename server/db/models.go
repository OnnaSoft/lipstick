package db

import "time"

type Domain struct {
	ID                       uint   `gorm:"primary_key"`
	Name                     string `gorm:"unique;not null"`
	ApiKey                   string `gorm:"not null"`
	AllowMultipleConnections bool   `gorm:"not null;default:true"`
}

type DailyConsumption struct {
	ID        uint      `gorm:"primary_key"`
	Domain    string    `gorm:"not null;index"`
	Date      time.Time `gorm:"type:date;not null"`
	Month     string    `gorm:"type:varchar(7);not null;index"`
	BytesUsed int64     `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
