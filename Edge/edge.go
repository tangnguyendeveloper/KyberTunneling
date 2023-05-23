package main

import (
	"fmt"

	TCPlib "github.com/tangnguyendeveloper/KyberTunneling/TCPLib"
)

func main() {
	const (
		cloud_host = "127.0.0.1"
		cloud_port = 443

		service_host = "127.0.0.1"
		service_port = 8000

		EdgeID = "ABCDEFGHIKLNM!@123456789"
	)

	fmt.Println("--------------------")
	fmt.Println("|    EDGE AGENT    |")
	fmt.Println("--------------------")

	edge_head := TCPlib.NewEdgeHead(cloud_host, cloud_port, service_host, service_port, EdgeID)
	edge_head.Start()
}
