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
	DomainID  uint      `gorm:"not null"`           // Foreign key to Domain
	Date      time.Time `gorm:"not null"`           // Date for the consumption record
	BytesUsed int64     `gorm:"not null;default:0"` // Total bytes used by the domain on this day
	CreatedAt time.Time // Automatically managed by GORM
	UpdatedAt time.Time // Automatically managed by GORM
}

// Relationship in Domain model
func (d *Domain) DailyConsumptions() []DailyConsumption {
	return []DailyConsumption{}
}
