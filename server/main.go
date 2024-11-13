package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"net/http"
	_ "net/http/pprof"

	"github.com/juliotorresmoreno/lipstick/server/admin"
	"github.com/juliotorresmoreno/lipstick/server/config"
	"github.com/juliotorresmoreno/lipstick/server/db"
	"github.com/juliotorresmoreno/lipstick/server/manager"
	"github.com/juliotorresmoreno/lipstick/server/proxy"
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

	proxyAddr := conf.Proxy.Address
	managerAddr := conf.Manager.Address

	db.Migrate()

	proxy := proxy.SetupProxy(proxyAddr, conf.TLS.CertificatePath, conf.TLS.KeyPath)
	manager := manager.SetupManager(proxy, managerAddr, conf.TLS.CertificatePath, conf.TLS.KeyPath)
	admin := admin.SetupAdmin(conf.Admin.Address)

	go manager.Listen()
	go admin.Listen()

	<-interrupt
	fmt.Println("Desconectando...")
}
