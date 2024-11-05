package db

type Domain struct {
	ID     uint   `gorm:"primary_key"`
	Name   string `gorm:"unique;not null"`
	ApiKey string `gorm:"not null"`
}
