package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"net/http"
	_ "net/http/pprof"

	"github.com/OnnaSoft/lipstick/helper"
	"github.com/OnnaSoft/lipstick/logger"
	"github.com/OnnaSoft/lipstick/server/admin"
	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/OnnaSoft/lipstick/server/db"
	"github.com/OnnaSoft/lipstick/server/manager"
	"github.com/OnnaSoft/lipstick/server/subscriptions"
)

func main() {
	if os.Getenv("DEBUG") == "true" {
		go func() {
			log.Println("Iniciando servidor pprof en :6060")
			log.Println(http.ListenAndServe(":6060", nil))
		}()
	}

	var conf, err = config.GetConfig()
	if err != nil {
		fmt.Println("Error al cargar la configuración", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	db.Migrate(conf.Database)
	subscriptions.GetRabbitMQManager(conf.RabbitMQ)

	tlsConfig := conf.TLS.GetTLSConfig()

	proxy := helper.NewListenerManagerTCP(conf.Proxy.Address, tlsConfig)
	manager := manager.SetupManager(tlsConfig)
	admin := admin.SetupAdmin(conf.Admin.Address)

	proxy.OnListen(func() { logger.Default.Info("Listening proxy on ", conf.Proxy.Address) })
	proxy.OnClose(func() { logger.Default.Info("Proxy closed") })
	proxy.OnTCPConn(func(c net.Conn) {
		go manager.HandleTCPConn(c)
	})
	proxy.OnHTTPConn(func(c net.Conn, req *http.Request) {
		go manager.HandleHTTPConn(c, req)
	})

	go manager.Listen()
	go admin.Listen()
	go proxy.ListenAndServe()
	<-interrupt
	fmt.Println("Desconectando...")
}
