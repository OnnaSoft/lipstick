package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/juliotorresmoreno/lipstick/manager"
	"github.com/juliotorresmoreno/lipstick/proxy"
	"github.com/juliotorresmoreno/lipstick/server/config"
	"github.com/juliotorresmoreno/lipstick/server/db"
)

func main() {
	var conf, err = config.GetConfig()
	if err != nil {
		fmt.Println("Error al cargar la configuración", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	proxyAddr := conf.Proxy.Addr
	managerAddr := conf.Manager.Addr

	db.Migrate()

	proxy := proxy.SetupProxy(proxyAddr, conf.Certs.Cert, conf.Certs.Key)
	manager := manager.SetupManager(conf.Keyword, proxy)

	go manager.Listen(managerAddr, conf.Certs.Cert, conf.Certs.Key)

	<-interrupt
	fmt.Println("Desconectando...")
}
