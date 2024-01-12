package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/juliotorresmoreno/turn/manager"
	"github.com/juliotorresmoreno/turn/proxy"
	"github.com/juliotorresmoreno/turn/server/config"
)

var conf, _ = config.GetConfig()

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	proxy := proxy.SetupProxy(conf.Proxy.Addr)
	manager := manager.SetupManager()

	go manager.Manage()
	go manager.Listen(conf.Manager.Addr)
	go manager.Forward()
	go proxy.Listen(manager)

	<-interrupt
	fmt.Println("Desconectando...")
}
