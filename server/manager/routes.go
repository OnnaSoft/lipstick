package manager

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/OnnaSoft/lipstick/server/db"
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

	r.GET("/traffic", router.getTraffic)
	r.GET("/health", router.health)
	r.GET("/", router.upgrade)

	manager.engine = r
}

func (r *router) getTraffic(c *gin.Context) {
	host := c.Request.Host
	domainName := strings.Split(host, ":")[0]

	fromParam := c.Query("from")
	toParam := c.Query("to")

	from, err := time.Parse("2006-01-02", fromParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' parameter"})
		return
	}

	to, err := time.Parse("2006-01-02", toParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' parameter"})
		return
	}

	to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	conf, err := config.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get config"})
		return
	}

	connection, err := db.GetConnection(conf.Database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to connect to database"})
		return
	}

	var consumptions []db.DailyConsumption
	connection.Where("domain = ? AND date >= ? AND date <= ?", domainName, from, to).Find(&consumptions)

	c.JSON(http.StatusOK, gin.H{
		"domain":       domainName,
		"from":         fromParam,
		"to":           toParam,
		"consumptions": consumptions,
	})
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
		c.String(http.StatusNotFound, "Domain "+domainName+" not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"domain": domain.Name,
	})
}

func (r *router) upgrade(c *gin.Context) {
	w := c.Writer
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	host := c.Request.Host
	domainName := strings.Split(host, ":")[0]
	if err != nil {
		log.Println("Unable to get domain name", err)
		conn.Close()
		return
	}

	domain, err := r.manager.authManager.GetDomain(domainName)
	if err != nil {
		log.Println("Unable to get domain", err)
		conn.Close()
		return
	}

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

	fmt.Fprint(rw, "HTTP/1.1 200 OK\r\n")
	fmt.Fprint(rw, "Content-Type: text/plain\r\n")
	fmt.Fprint(rw, "\r\n")
	if err := rw.Flush(); err != nil {
		log.Println("Error flushing headers:", err)
		return
	}
	notification := &ProxyNotificationConn{
		Domain:                   domain.Name,
		conn:                     conn,
		ReadWriter:               rw,
		AllowMultipleConnections: domain.AllowMultipleConnections,
	}

	hub.registerProxyNotificationConn <- notification
}
