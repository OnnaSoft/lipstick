package manager

import (
	"log"
	"net/http"
	"strings"

	"github.com/OnnaSoft/lipstick/helper"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type router struct {
	manager *Manager
}

func configureRouter(manager *Manager) {
	router := &router{manager: manager}
	r := gin.New()

	r.GET("/health", router.health)
	r.GET("/", router.upgrade)

	manager.engine = r
}

func (r *router) health(c *gin.Context) {
	host := c.Request.Host
	domainName := strings.Split(host, ":")[0]

	if domain, ok := r.manager.GetHub(domainName); ok {
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

	var hub *NetworkHub
	hub, ok := r.manager.GetHub(domain.Name)
	if !ok {
		hub = NewNetworkHub(
			domain.Name,
			r.manager.trafficManager,
			64*1024,
		)
		r.manager.AddHub(domain.Name, hub)
		go hub.listen()
	}

	notification := &ProxyNotificationConn{
		Domain:                   domain.Name,
		Conn:                     wsConn,
		AllowMultipleConnections: domain.AllowMultipleConnections,
	}
	hub.registerProxyNotificationConn <- notification
}
