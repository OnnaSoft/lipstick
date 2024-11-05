package auth

type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type Domain struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	ApiKey string `json:"apiKey"`
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
