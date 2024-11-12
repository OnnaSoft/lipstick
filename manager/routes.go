package manager

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/juliotorresmoreno/lipstick/helper"
)

type router struct {
	manager *Manager
}

func configureRouter(manager *Manager) {
	router := &router{manager: manager}
	r := gin.New()

	r.GET("/health", router.health)
	r.GET("/ws", router.upgrade)
	r.GET("/ws/:uuid", router.request)

	manager.engine = r
}

func (r *router) health(c *gin.Context) {
	host := c.Request.Host
	domainName := strings.Split(host, ":")[0]

	fmt.Println("Domain:", domainName)

	if domain, ok := r.manager.hubs[domainName]; ok {
		fmt.Println("Domain found")
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

func (r *router) upgrade(c *gin.Context) {
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Unable to upgrade connection", err)
		return
	}

	domainName, err := helper.GetDomainName(wsConn.NetConn())
	if err != nil {
		log.Println("Unable to get domain name", err)
		wsConn.Close()
		return
	}

	domain, err := r.manager.authManager.GetDomain(domainName)
	if err != nil {
		log.Println("Unable to get domain", err)
		wsConn.Close()
		return
	}

	r.manager.registerDomain <- &websocketConn{
		Domain:        domain.Name,
		Conn:          wsConn,
		AllowMultiple: true,
	}
}

func (r *router) request(c *gin.Context) {
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Unable to upgrade connection", err)
		return
	}

	domainName, err := helper.GetDomainName(wsConn.NetConn())
	if err != nil {
		log.Println("Unable to get domain name", err)
		wsConn.Close()
		return
	}

	uuid, ok := c.Params.Get("uuid")
	if !ok {
		return
	}
	domain, ok := r.manager.hubs[domainName]
	if !ok {
		return
	}

	domain.serverRequests <- &request{uuid: uuid, conn: wsConn}
}
