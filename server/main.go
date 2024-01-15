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

	Listen(managerAddr, proxyAddr, conf.Keyword)

	<-interrupt
	fmt.Println("Desconectando...")
}

func Listen(managerAddr, proxyAddr string, keyword string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	proxy := proxy.SetupProxy(proxyAddr)
	manager := manager.SetupManager(keyword)

	go manager.Manage()
	go manager.Listen(managerAddr)
	go manager.Forward()
	go proxy.Listen(manager)

	<-interrupt
	fmt.Println("Desconectando...")
}
