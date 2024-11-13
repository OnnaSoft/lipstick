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
	tx := p.db.Where("name = ?", domain).First(result)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &Domain{
		ID:                       result.ID,
		Name:                     result.Name,
		ApiKey:                   result.ApiKey,
		AllowMultipleConnections: result.AllowMultipleConnections,
	}, nil
}

func (p *PostgresAuthManager) AddDomain(domain *Domain) error {
	tx := p.db.Create(&db.Domain{
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
