package main

import (
	"clustering"
	"conf"
	"distributor"
	"fmt"
	"net/http"
	"schedule"
)

func main() {
	if conf.ReadFlags() {
		conf.Configure()

		fmt.Println("Setting network...")
		success, conn := clustering.Network()

		if !success {
			fmt.Println("Server going down...")
			return
		}

		fmt.Println("\nStarting scheduler and distributor...")

		go distributor.ProcessMessageQueue()
		schedule.InitScheduler(conn)
		routes()

		//http.ListenAndServeTLS(conf.GetPort(), "/root/sellyx/certs/cert.pem",
		//	"/root/sellyx/certs/key.key", nil)

		http.ListenAndServe(conf.GetPort(), nil)
	}

}
