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
	Domain    string    `gorm:"not null;index"`     // Domain name with an index
	Date      time.Time `gorm:"type:date;not null"` // Date only (no time) with an index
	BytesUsed int64     `gorm:"not null;default:0"` // Total bytes used by the domain on this day
	CreatedAt time.Time
	UpdatedAt time.Time
}
