package auth

type Domain struct {
	ID                       uint   `json:"id"`
	Name                     string `json:"name"`
	ApiKey                   string `json:"apiKey"`
	AllowMultipleConnections bool   `json:"allowMultipleConnections"`
}

type AuthManager interface {
	GetDomains() ([]*Domain, error)
	GetDomain(domain string) (*Domain, error)
	AddDomain(domain *Domain) error
	UpdateDomain(domain *Domain) error
	DelDomain(id uint) error
}

func MakeAuthManager() AuthManager {
	manager := NewPostgresAuthManager()

	return manager
}
