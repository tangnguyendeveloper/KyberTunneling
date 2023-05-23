package main

import (
	"fmt"

	TCPlib "github.com/tangnguyendeveloper/KyberTunneling/TCPLib"
)

func main() {

	const (
		queue_capacity = 10000
		bind_host      = "127.0.0.1"
		bind_port      = 443
	)

	fmt.Println("----------------------")
	fmt.Println("|    CLOUD MASTER    |")
	fmt.Println("----------------------")

	cloud_fw := TCPlib.NewCloudForwarder(bind_host, bind_port)
	cloud_fw.Start(queue_capacity)
}
