package main

import (
	"fmt"

	TCPlib "github.com/tangnguyendeveloper/KyberTunneling/TCPLib"
)

func main() {
	const (
		cloud_host = "27.71.25.238"
		cloud_port = 443

		bind_host = "127.0.0.1"
		bind_port = 80

		EdgeID = "ABCDEFGHIKLNM!@123456789"

		queue_capacity = 1024
	)

	fmt.Println("-------------------")
	fmt.Println("|    APP AGENT    |")
	fmt.Println("-------------------")

	app_head := TCPlib.NewAppHead(bind_host, bind_port, cloud_host, cloud_port, EdgeID)
	app_head.Start(queue_capacity)
}
