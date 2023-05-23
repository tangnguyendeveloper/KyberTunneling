package TCPlib

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

type CloudForwarder struct {
	BindHost string
	BindPort uint

	mapper  map[string]net.Conn
	session map[string]chan net.Conn
}

func NewCloudForwarder(host string, port uint) *CloudForwarder {
	return &CloudForwarder{
		BindHost: host,
		BindPort: port,
		mapper:   make(map[string]net.Conn),
		session:  make(map[string]chan net.Conn),
	}
}

func (c_fw *CloudForwarder) Start(queue_capacity uint) {
	listener := NewTCPListener(c_fw.BindHost, c_fw.BindPort, queue_capacity)
	go listener.Start()

	for {
		conn, ok := <-listener.QueueAccepted
		if !ok {
			time.Sleep(time.Millisecond)
			continue
		}
		go c_fw.handleConnection(conn)
	}
}

func (c_fw *CloudForwarder) handleConnection(conn net.Conn) {
	cmd_msg, err := ReceiveCommandMessage(conn)
	if err != nil {
		log.Printf("ERROR: ReceiveConnectMessage, %s\n", err)
		conn.Close()
		return
	}

	switch cmd_msg.Command {
	case EDGE_CONNECT, EDGE_ALIVE:
		edgeID := string(cmd_msg.Parameter)
		c_fw.mapper[edgeID] = conn
		if cmd_msg.Command == EDGE_ALIVE {
			log.Printf("INFO: EDGE_ALIVE with id %x\n", edgeID)
			return
		}
		log.Printf("INFO: EDGE_CONNECT with id %x\n", edgeID)
		sendSuccessACK(conn, "200 ok")

	case APP_CONNECT:
		edge_conn, ok := c_fw.mapper[string(cmd_msg.Parameter)]
		if !ok {
			sendErrorACK(conn, "404 Not Found")
		}
		c_fw.forward(edge_conn, conn)

	case EDGE_OPEN_LINK_RESPONSE:
		ch, ok := c_fw.session[string(cmd_msg.Parameter)]
		if !ok {
			sendErrorACK(conn, "400 Bad Request")
		}
		ch <- conn

	default:
		conn.Close()
		return
	}
}

func (c_fw *CloudForwarder) forward(edge_conn net.Conn, app_conn net.Conn) {
	sessionID := make([]byte, 8)
	rand.Read(sessionID)

	c_fw.session[string(sessionID)] = make(chan net.Conn, 1)

	ok := sendOpenLinkRequest(edge_conn, sessionID)
	if !ok {
		sendErrorACK(app_conn, "502 Bad Gateway")
		return
	}

	var s_conn net.Conn

	for {
		s_conn, ok = <-c_fw.session[string(sessionID)]
		if !ok {
			time.Sleep(time.Millisecond)
			continue
		}
		delete(c_fw.session, string(sessionID))
		break
	}

	sendSuccessACK(s_conn, "200 ok, link opened")
	sendSuccessACK(app_conn, "200 ok, link opened")

	defer s_conn.Close()
	defer app_conn.Close()

	go func() {
		if _, err := io.Copy(s_conn, app_conn); err != nil {
			log.Printf("Failed forwarding to Edge: %s\n", err)
		}
	}()

	if _, err := io.Copy(app_conn, s_conn); err != nil {
		log.Printf("Failed forwarding to App: %s\n", err)
	}

}

func sendOpenLinkRequest(conn net.Conn, sessionID []byte) bool {
	var buffer bytes.Buffer

	buffer.WriteByte(EDGE_OPEN_LINK_REQUEST)

	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, 8)
	buffer.Write(length)

	buffer.Write(sessionID)

	_, err := conn.Write(buffer.Bytes())
	if err != nil {
		log.Printf("ERROR: sendOpenLinkRequest, %s\n", err)
		return false
	}
	return true
}

func sendErrorACK(conn net.Conn, message string) {
	var buffer bytes.Buffer

	buffer.WriteByte(ERROR_ACK)

	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(message)))
	buffer.Write(length)

	buffer.WriteString(message)

	conn.Write(buffer.Bytes())

	conn.Close()
}

func sendSuccessACK(conn net.Conn, message string) {
	var buffer bytes.Buffer

	buffer.WriteByte(SUCCESS_ACK)

	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(message)))
	buffer.Write(length)

	buffer.WriteString(message)

	conn.Write(buffer.Bytes())
}
