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
		cacheTTL: 5 * time.Minute, // Configura el TTL de la caché
	}
}

// Internal method to get cached data or fallback to the database
func (p *PostgresAuthManager) getCached(key string, fallback func() (interface{}, error)) (interface{}, error) {
	if entry, found := p.cache.Load(key); found {
		cached := entry.(cacheEntry)
		if time.Since(cached.timestamp) < p.cacheTTL {
			return cached.data, nil
		}
		// Invalidate expired entry
		p.cache.Delete(key)
	}

	// Fallback to the database
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	// Double-check to avoid redundant queries
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

// GetDomains retrieves all domains with caching
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

// GetDomain retrieves a single domain by name with caching
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

// AddDomain adds a new domain
func (p *PostgresAuthManager) AddDomain(domain *Domain) error {
	tx := p.db.Create(&db.Domain{
		Name:                     domain.Name,
		ApiKey:                   domain.ApiKey,
		AllowMultipleConnections: domain.AllowMultipleConnections,
	})
	if tx.Error != nil {
		return tx.Error
	}

	// Invalidate cache for all domains
	p.cache.Delete("all_domains")
	return nil
}

// UpdateDomain updates an existing domain
func (p *PostgresAuthManager) UpdateDomain(domain *Domain) error {
	tx := p.db.Model(&db.Domain{}).Where("id = ?", domain.ID).Updates(map[string]interface{}{
		"name":                       domain.Name,
		"api_key":                    domain.ApiKey,
		"allow_multiple_connections": domain.AllowMultipleConnections,
	})
	if tx.Error != nil {
		return tx.Error
	}

	// Invalidate cache for the specific domain and all domains
	p.cache.Delete("domain_" + domain.Name)
	p.cache.Delete("all_domains")
	return nil
}

// DelDomain deletes a domain by ID
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

	// Invalidate cache for the specific domain and all domains
	p.cache.Delete("domain_" + result.Name)
	p.cache.Delete("all_domains")
	return nil
}
