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
	"github.com/OnnaSoft/lipstick/server/admin"
	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/OnnaSoft/lipstick/server/db"
	"github.com/OnnaSoft/lipstick/server/manager"
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

	tlsConfig := conf.TLS.GetTLSConfig()

	proxy := helper.NewListenerManagerTCP(conf.Proxy.Address, tlsConfig)
	manager := manager.SetupManager(tlsConfig)
	admin := admin.SetupAdmin(conf.Admin.Address)

	proxy.OnListen(func() { log.Println("Listening login on", conf.Proxy.Address) })
	proxy.OnClose(func() { log.Println("Proxy closed") })
	proxy.OnTCPConn(func(c net.Conn) {
		go manager.HandleTCPConn(c)
	})
	proxy.OnHTTPConn(func(c net.Conn) {
		go manager.HandleHTTPConn(c)
	})

	go manager.Listen()
	go admin.Listen()
	go proxy.ListenAndServe()
	<-interrupt
	fmt.Println("Desconectando...")
}
