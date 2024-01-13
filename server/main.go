package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/juliotorresmoreno/turn/manager"
	"github.com/juliotorresmoreno/turn/proxy"
	"github.com/juliotorresmoreno/turn/server/config"
)

func main() {
	var conf, _ = config.GetConfig()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	proxyAddr := conf.Proxy.Addr
	managerAddr := conf.Manager.Addr

	Listen(managerAddr, proxyAddr)

	<-interrupt
	fmt.Println("Desconectando...")
}

func Listen(managerAddr, proxyAddr string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	proxy := proxy.SetupProxy(proxyAddr)
	manager := manager.SetupManager()

	go manager.Manage()
	go manager.Listen(managerAddr)
	go manager.Forward()
	go proxy.Listen(manager)

	<-interrupt
	fmt.Println("Desconectando...")
}
