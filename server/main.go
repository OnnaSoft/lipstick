package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/juliotorresmoreno/lipstick/manager"
	"github.com/juliotorresmoreno/lipstick/proxy"
	"github.com/juliotorresmoreno/lipstick/server/config"
)

func main() {
	var conf, _ = config.GetConfig()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	proxyAddr := conf.Proxy.Addr
	managerAddr := conf.Manager.Addr

	proxy := proxy.SetupProxy(proxyAddr, conf.Certs.Cert, conf.Certs.Key)
	manager := manager.SetupManager(conf.Keyword, proxy)

	go manager.Listen(managerAddr, conf.Certs.Cert, conf.Certs.Key)

	<-interrupt
	fmt.Println("Desconectando...")
}
