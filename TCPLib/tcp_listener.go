package TCPlib

import (
	"fmt"
	"log"
	"net"
)

type TCPListener struct {
	Host          string
	Port          uint
	QueueAccepted chan net.Conn
}

func NewTCPListener(host string, port uint, queue_capacity uint) *TCPListener {
	return &TCPListener{
		Host:          host,
		Port:          port,
		QueueAccepted: make(chan net.Conn, queue_capacity),
	}
}

func (listener *TCPListener) Start() {
	var address string = fmt.Sprintf("%s:%d", listener.Host, listener.Port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("ERROR: TCP Listen at %s -> %s", address, err)
	}
	log.Printf("INFO: TCP Listening at %s \n", address)

	defer ln.Close()
	defer close(listener.QueueAccepted)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("ERROR: TCP Accept -> %s \n", err)
			continue
		}
		listener.QueueAccepted <- conn
	}
}
