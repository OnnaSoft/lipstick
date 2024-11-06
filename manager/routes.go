package manager

import (
	"log"
	"net/http"

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
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

	r.manager.registerWebsocketConn <- &websocketConn{
		Domain: domain.Name,
		Conn:   wsConn,
	}
}

func (r *router) request(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Unable to upgrade connection", err)
		return
	}

	uuid, ok := c.Params.Get("uuid")
	if !ok {
		return
	}
	r.manager.request <- &request{uuid: uuid, conn: conn}
}
