package manager

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/juliotorresmoreno/lipstick/helper"
	"github.com/juliotorresmoreno/lipstick/server/auth"
	"github.com/juliotorresmoreno/lipstick/server/config"
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

	r.GET("/users", router.getUsers)
	r.GET("/users/:id", router.getUser)
	r.POST("/users", router.addUser)
	r.PATCH("/users/:id", router.updateUser)
	r.DELETE("/users/:id", router.deleteUser)

	r.GET("/domains", router.getDomains)
	r.POST("/domains", router.addDomain)

	manager.engine = r
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

	domains, err := r.manager.authManager.GetDomains()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get domains"})
		return
	}

	c.JSON(http.StatusOK, domains)
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

	if err := r.manager.authManager.AddDomain(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to add domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (r *router) getUsers(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	users, err := r.manager.authManager.GetUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (r *router) getUser(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := helper.ParseUint(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	user, err := r.manager.authManager.GetUser(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (r *router) addUser(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user := &auth.User{}
	if err := c.BindJSON(user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.manager.authManager.AddUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to add user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (r *router) updateUser(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user := &auth.User{}
	if err := c.BindJSON(user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.ID = helper.ParseUint(c.Param("id"))
	if user.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	if err := r.manager.authManager.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (r *router) deleteUser(c *gin.Context) {
	if !isAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := helper.ParseUint(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	if err := r.manager.authManager.DelUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

	user, domain, err := r.manager.authManager.GetUserByDomain(domainName)
	if err != nil {
		log.Println("Unable to get user by domain", err)
		wsConn.Close()
		return
	}

	connections := r.manager.userConnections[user.ID]
	if connections >= user.Limit {
		log.Println("User limit exceeded", user.ID)
		wsConn.Close()
		return
	}

	r.manager.registerWebsocketConn <- &websocketConn{
		UserID: user.ID,
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
