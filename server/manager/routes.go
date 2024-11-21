package manager

import (
	"log"
	"net/http"
	"strings"

	"github.com/OnnaSoft/lipstick/helper"
	"github.com/gin-gonic/gin"
)

type router struct {
	manager *Manager
}

func configureRouter(manager *Manager) {
	router := &router{manager: manager}
	r := gin.New()

	r.GET("/health", router.health)
	r.GET("/", router.strem)

	manager.engine = r
}

func (r *router) health(c *gin.Context) {
	host := c.Request.Host
	domainName := strings.Split(host, ":")[0]

	if domain, ok := r.manager.hubs[domainName]; ok {
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"domain":     domain.HubName,
			"data_usage": domain.totalDataTransferred,
		})
		return
	}

	domain, err := r.manager.authManager.GetDomain(domainName)
	if err != nil {
		c.String(http.StatusNotFound, "Domain not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"domain": domain.Name,
	})
}

func (r *router) strem(c *gin.Context) {
	rw, err := NewHttpReadWriter(c.Writer)
	if err != nil {
		log.Println("Unable to hijack connection", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	domainName, err := helper.GetDomainName(rw.conn)
	if err != nil {
		log.Println("Unable to get domain name", err)
		rw.Close()
		return
	}

	domain, err := r.manager.authManager.GetDomain(domainName)
	if err != nil {
		log.Println("Unable to get domain", err)
		rw.Close()
		return
	}

	rw.WriteString("HTTP/1.1 200 OK\r\n")
	rw.WriteString("Content-Type: text/plain\r\n")
	rw.WriteString("Connection: keep-alive\r\n")
	rw.WriteString("Transfer-Encoding: chunked\r\n")
	rw.WriteString("\r\n")
	rw.Flush()

	r.manager.registerProxyNotificationConn <- &ProxyNotificationConn{
		Domain:                   domain.Name,
		HttpReadWriter:           rw,
		AllowMultipleConnections: domain.AllowMultipleConnections,
	}
}
