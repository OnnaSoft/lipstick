package db

type User struct {
	ID       uint   `gorm:"primary_key"`
	Username string `gorm:"unique;not null"`
	Limit    uint   `gorm:"not null"`
}

type Domain struct {
	ID     uint   `gorm:"primary_key"`
	UserID uint   `gorm:"not null;index"`
	User   User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:UserID"`
	Name   string `gorm:"unique;not null"`
	ApiKey string `gorm:"unique;not null"`
}
