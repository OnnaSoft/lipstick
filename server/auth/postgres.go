package auth

import (
	"log"

	"github.com/jinzhu/gorm"
	"github.com/juliotorresmoreno/lipstick/server/db"
)

type PostgresAuthManager struct {
	db *gorm.DB
}

func NewPostgresAuthManager() AuthManager {
	conn, err := db.GetConnection()
	if err != nil {
		log.Fatal(err)
	}

	return &PostgresAuthManager{
		db: conn,
	}
}

func (p *PostgresAuthManager) GetUsers() ([]*User, error) {
	users := []*db.User{}
	if tx := p.db.Find(&users); tx.Error != nil {
		return nil, tx.Error
	}

	result := make([]*User, len(users))
	for i, user := range users {
		result[i] = &User{
			ID:       user.ID,
			Username: user.Username,
			Limit:    user.Limit,
		}
	}

	return result, nil
}

func (p *PostgresAuthManager) GetUser(id uint) (*User, error) {
	user := &db.User{}
	if tx := p.db.First(user, id); tx.Error != nil {
		return nil, tx.Error
	}

	return &User{
		ID:       user.ID,
		Username: user.Username,
		Limit:    user.Limit,
	}, nil
}

func (p *PostgresAuthManager) AddUser(user *User) error {
	tx := p.db.Create(&db.User{
		Username: user.Username,
		Limit:    user.Limit,
	})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (p *PostgresAuthManager) UpdateUser(user *User) error {
	tx := p.db.Model(&db.User{}).Where("id = ?", user.ID).Update(db.User{
		Username: user.Username,
		Limit:    user.Limit,
	})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (p *PostgresAuthManager) DelUser(id uint) error {
	tx := p.db.Delete(&db.User{}, id)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (p *PostgresAuthManager) GetDomains() ([]*Domain, error) {
	domains := []*db.Domain{}
	if tx := p.db.Find(&domains); tx.Error != nil {
		return nil, tx.Error
	}

	result := make([]*Domain, len(domains))
	for i, domain := range domains {
		result[i] = &Domain{
			ID:     domain.ID,
			Name:   domain.Name,
			ApiKey: domain.ApiKey,
		}
	}

	return result, nil
}

func (p *PostgresAuthManager) GetDomain(domain string) (*Domain, error) {
	result := &db.Domain{}
	tx := p.db.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Limit(1)
	}).Where("name = ?", domain).First(result)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &Domain{
		ID:     result.ID,
		Name:   result.Name,
		ApiKey: result.ApiKey,
		UserID: result.UserID,
		User: &User{
			ID:       result.User.ID,
			Username: result.User.Username,
			Limit:    result.User.Limit,
		},
	}, nil
}

func (p *PostgresAuthManager) AddDomain(domain *Domain) error {
	tx := p.db.Create(&db.Domain{
		UserID: domain.UserID,
		Name:   domain.Name,
		ApiKey: domain.ApiKey,
	})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (p *PostgresAuthManager) UpdateDomain(domain *Domain) error {
	tx := p.db.Model(&db.Domain{}).Where("id = ?", domain.ID).Update(domain)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (p *PostgresAuthManager) DelDomain(id uint) error {
	tx := p.db.Delete(&db.Domain{}, id)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
