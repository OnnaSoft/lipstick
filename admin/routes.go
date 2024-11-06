package admin

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/juliotorresmoreno/lipstick/server/auth"
	"github.com/juliotorresmoreno/lipstick/server/config"
)

type router struct {
	admin *Admin
}

func configureRouter(admin *Admin) {
	router := &router{admin: admin}
	r := gin.New()

	r.GET("/health", router.health)

	r.GET("/domains", router.getDomains)
	r.GET("/domains/:domain_name", router.getDomain)
	r.POST("/domains", router.addDomain)
	r.PATCH("/domains/:domain_name", router.updateDomain)
	r.DELETE("/domains/:domain_name", router.deleteDomain)

	admin.engine = r
}

func isAuthorized(c *gin.Context) bool {
	conf, err := config.GetConfig()
	if err != nil {
		log.Println("Unable to get config", err)
		return false
	}
	authorization := c.Request.Header.Get("Authorization")
	return authorization == conf.Keyword
}

func (r *router) getDomains(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	domains, err := r.admin.authManager.GetDomains()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get domains"})
		return
	}

	c.JSON(http.StatusOK, domains)
}

func (r *router) getDomain(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	domain_name := c.Param("domain_name")
	domain, err := r.admin.authManager.GetDomain(domain_name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get domain"})
		return
	}

	c.JSON(http.StatusOK, domain)
}

func (r *router) addDomain(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	domain := &auth.Domain{}
	if err := c.BindJSON(domain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.admin.authManager.AddDomain(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to add domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (r *router) updateDomain(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	domain := &auth.Domain{}
	if err := c.BindJSON(domain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	domain_name := c.Param("domain_name")
	record, err := r.admin.authManager.GetDomain(domain_name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get domain"})
		return
	}

	domain.ID = record.ID
	if err := r.admin.authManager.UpdateDomain(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to update domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (r *router) deleteDomain(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	domain_name := c.Param("domain_name")
	record, err := r.admin.authManager.GetDomain(domain_name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get domain"})
		return
	}

	if err := r.admin.authManager.DelDomain(record.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to delete domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (r *router) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
