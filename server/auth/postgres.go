package auth

import (
	"log"
	"sync"
	"time"

	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/OnnaSoft/lipstick/server/db"
	"gorm.io/gorm"
)

type PostgresAuthManager struct {
	db         *gorm.DB
	cache      sync.Map
	cacheTTL   time.Duration
	cacheMutex sync.Mutex
}

type cacheEntry struct {
	data      interface{}
	timestamp time.Time
}

func NewPostgresAuthManager() AuthManager {
	conf, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	conn, err := db.GetConnection(conf.Database)
	if err != nil {
		log.Fatal(err)
	}

	return &PostgresAuthManager{
		db:       conn,
		cacheTTL: 5 * time.Minute,
	}
}

func (p *PostgresAuthManager) getCached(key string, fallback func() (interface{}, error)) (interface{}, error) {
	if entry, found := p.cache.Load(key); found {
		cached := entry.(cacheEntry)
		if time.Since(cached.timestamp) < p.cacheTTL {
			return cached.data, nil
		}
		p.cache.Delete(key)
	}

	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	if entry, found := p.cache.Load(key); found {
		cached := entry.(cacheEntry)
		if time.Since(cached.timestamp) < p.cacheTTL {
			return cached.data, nil
		}
		p.cache.Delete(key)
	}

	data, err := fallback()
	if err == nil {
		p.cache.Store(key, cacheEntry{
			data:      data,
			timestamp: time.Now(),
		})
	}
	return data, err
}

func (p *PostgresAuthManager) GetDomains() ([]*Domain, error) {
	key := "all_domains"
	data, err := p.getCached(key, func() (interface{}, error) {
		domains := []*db.Domain{}
		if tx := p.db.Find(&domains); tx.Error != nil {
			return nil, tx.Error
		}

		result := make([]*Domain, len(domains))
		for i, domain := range domains {
			result[i] = &Domain{
				ID:                       domain.ID,
				Name:                     domain.Name,
				ApiKey:                   domain.ApiKey,
				AllowMultipleConnections: domain.AllowMultipleConnections,
			}
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return data.([]*Domain), nil
}

func (p *PostgresAuthManager) GetDomain(domain string) (*Domain, error) {
	key := "domain_" + domain
	data, err := p.getCached(key, func() (interface{}, error) {
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
	})
	if err != nil {
		return nil, err
	}
	return data.(*Domain), nil
}

func (p *PostgresAuthManager) AddDomain(domain *Domain) error {
	tx := p.db.Create(&db.Domain{
		Name:                     domain.Name,
		ApiKey:                   domain.ApiKey,
		AllowMultipleConnections: domain.AllowMultipleConnections,
	})
	if tx.Error != nil {
		return tx.Error
	}

	p.cache.Delete("all_domains")
	return nil
}

func (p *PostgresAuthManager) UpdateDomain(domain *Domain) error {
	tx := p.db.Model(&db.Domain{}).Where("id = ?", domain.ID).Updates(map[string]interface{}{
		"name":                       domain.Name,
		"api_key":                    domain.ApiKey,
		"allow_multiple_connections": domain.AllowMultipleConnections,
	})
	if tx.Error != nil {
		return tx.Error
	}

	p.cache.Delete("domain_" + domain.Name)
	p.cache.Delete("all_domains")
	return nil
}

func (p *PostgresAuthManager) DelDomain(id uint) error {
	result := &db.Domain{}
	tx := p.db.First(result, id)
	if tx.Error != nil {
		return tx.Error
	}

	tx = p.db.Delete(&db.Domain{}, id)
	if tx.Error != nil {
		return tx.Error
	}

	p.cache.Delete("domain_" + result.Name)
	p.cache.Delete("all_domains")
	return nil
}
