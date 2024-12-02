package manager

import (
	"net/http"
	"strings"
	"time"

	"github.com/OnnaSoft/lipstick/logger"
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
		logger.Default.Error("Invalid 'from' parameter:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' parameter"})
		return
	}

	to, err := time.Parse("2006-01-02", toParam)
	if err != nil {
		logger.Default.Error("Invalid 'to' parameter:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' parameter"})
		return
	}

	to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	conf, err := config.GetConfig()
	if err != nil {
		logger.Default.Error("Error getting configuration:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get config"})
		return
	}

	connection, err := db.GetConnection(conf.Database)
	if err != nil {
		logger.Default.Error("Error connecting to database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to connect to database"})
		return
	}

	var consumptions []db.DailyConsumption
	connection.Where("domain = ? AND date >= ? AND date <= ?", domainName, from, to).Find(&consumptions)

	logger.Default.Info("Traffic data retrieved for domain:", domainName)
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
		logger.Default.Info("Health check for active domain:", domainName)
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"domain":     domain.HubName,
			"data_usage": domain.totalDataTransferred,
		})
		return
	}

	domain, err := r.manager.authManager.GetDomain(domainName)
	if err != nil {
		logger.Default.Error("Domain not found in health check:", domainName, "Error:", err)
		c.String(http.StatusNotFound, "Domain "+domainName+" not found")
		return
	}

	logger.Default.Info("Health check for inactive domain:", domainName)
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"domain": domain.Name,
	})
}

func (r *router) upgrade(c *gin.Context) {
	w := c.Writer
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		logger.Default.Error("Hijacking not supported on writer")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		logger.Default.Error("Failed to hijack connection:", err)
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}

	host := c.Request.Host
	domainName := strings.Split(host, ":")[0]
	domain, err := r.manager.authManager.GetDomain(domainName)
	if err != nil {
		logger.Default.Error("Unable to get domain:", domainName, "Error:", err)
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
		logger.Default.Info("New hub created for domain:", domain.Name)
	}

	rw.WriteString("HTTP/1.1 200 OK\r\n")
	rw.WriteString("Content-Type: text/plain\r\n")
	rw.WriteString("\r\n")
	if err := rw.Flush(); err != nil {
		logger.Default.Error("Error flushing headers for upgrade:", err)
		return
	}

	logger.Default.Info("Connection upgraded for domain:", domain.Name)
	notification := &ProxyNotificationConn{
		Domain:                   domain.Name,
		conn:                     conn,
		ReadWriter:               rw,
		AllowMultipleConnections: domain.AllowMultipleConnections,
	}

	hub.registerProxyNotificationConn <- notification
}
