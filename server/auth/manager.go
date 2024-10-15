package auth

type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Limit    uint   `json:"limit"`
}

type Domain struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	ApiKey string `json:"apiKey"`
	UserID uint   `json:"userId"`
	User   *User  `json:"user"`
}

type AuthManager interface {
	GetUsers() ([]*User, error)
	GetUser(id uint) (*User, error)
	AddUser(user *User) error
	UpdateUser(user *User) error
	DelUser(id uint) error

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
