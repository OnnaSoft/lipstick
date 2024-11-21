package admin

import (
	"log"

	"github.com/OnnaSoft/lipstick/server/auth"
	"github.com/gin-gonic/gin"
)

type Admin struct {
	engine      *gin.Engine
	authManager auth.AuthManager
	addr        string
}

func SetupAdmin(addr string) *Admin {
	gin.SetMode(gin.ReleaseMode)

	admin := &Admin{
		authManager: auth.MakeAuthManager(),
		addr:        addr,
	}

	configureRouter(admin)

	return admin
}

func (a *Admin) Listen() {
	log.Println("Listening admin on", a.addr)
	a.engine.Run(a.addr)
}
