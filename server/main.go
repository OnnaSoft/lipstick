package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/juliotorresmoreno/lipstick/server/admin"
	"github.com/juliotorresmoreno/lipstick/server/config"
	"github.com/juliotorresmoreno/lipstick/server/db"
	"github.com/juliotorresmoreno/lipstick/server/manager"
	"github.com/juliotorresmoreno/lipstick/server/proxy"
)

func main() {
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
