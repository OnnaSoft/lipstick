package admin

import (
	"log"
	"net/http"

	"github.com/OnnaSoft/lipstick/server/auth"
	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/gin-gonic/gin"
)

type router struct {
	admin *Admin
}

func configureRouter(admin *Admin) {
	router := &router{admin: admin}
	r := gin.New()

	r.GET("/health", router.health)

	r.GET("/domains", router.getDomains)
	const domainNamePath = "/domains/:domainName"

	r.GET(domainNamePath, router.getDomain)
	r.POST("/domains", router.addDomain)
	r.PATCH(domainNamePath, router.updateDomain)
	r.DELETE(domainNamePath, router.deleteDomain)

	admin.engine = r
}

func isAuthorized(c *gin.Context) bool {
	conf, err := config.GetConfig()
	if err != nil {
		log.Println("Unable to get config", err)
		return false
	}
	authorization := c.Request.Header.Get("Authorization")
	return authorization == conf.AdminSecretKey
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

	domainName := c.Param("domainName")
	domain, err := r.admin.authManager.GetDomain(domainName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to get domain"})
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

	domain := map[string]interface{}{}
	if err := c.BindJSON(&domain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	domainName := c.Param("domainName")
	record, err := r.admin.authManager.GetDomain(domainName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get domain"})
		return
	}

	if _, ok := domain["apiKey"]; ok {
		record.ApiKey = domain["apiKey"].(string)
	}
	if _, ok := domain["allowMultipleConnections"]; ok {
		record.AllowMultipleConnections = domain["allowMultipleConnections"].(bool)
	}

	if err := r.admin.authManager.UpdateDomain(record); err != nil {
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

	domainName := c.Param("domainName")
	record, err := r.admin.authManager.GetDomain(domainName)
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
