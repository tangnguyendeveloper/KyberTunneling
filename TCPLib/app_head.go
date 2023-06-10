package TCPlib

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type AppHead struct {
	BindHost string
	BindPort uint

	CloudHost string
	CloudPort uint

	EdgeID string
}

func NewAppHead(BindHost string, BindPort uint, CloudHost string, CloudPort uint, EdgeID string) *AppHead {
	return &AppHead{
		CloudHost: CloudHost, CloudPort: CloudPort, BindHost: BindHost, BindPort: BindPort, EdgeID: EdgeID,
	}
}

func (ac *AppHead) Start(queue_capacity uint) {
	listener := NewTCPListener(ac.BindHost, ac.BindPort, queue_capacity)
	go listener.Start()

	for {
		conn, ok := <-listener.QueueAccepted
		if !ok {
			time.Sleep(time.Millisecond)
			continue
		}
		go ac.handleConnection(conn)
	}
}

func (ac *AppHead) handleConnection(conn net.Conn) {
	session_conn := ac.dialCloud()
	if session_conn == nil {
		conn.Close()
		return
	}

	ok := ac.sendConnect(session_conn)
	if !ok {
		conn.Close()
		return
	}

	appForwarding(session_conn, conn)

}

func appForwarding(session_conn net.Conn, client_conn net.Conn) {

	key := []byte("1234567890abohtdgetonahuytekhu@%")

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("ERROR: Failed to create AES cipher: %s\n", err)
		return
	}

	iv := make([]byte, aes.BlockSize)
	stream := cipher.NewCTR(block, iv)

	defer session_conn.Close()
	defer client_conn.Close()

	go func() {

		decryptedReader := cipher.StreamReader{S: stream, R: session_conn}

		buffer := make([]byte, MAX_TCP_BUFFER)

		if _, err := io.CopyBuffer(client_conn, decryptedReader, buffer); err != nil {
			log.Printf("Failed forwarding to App: %s\n", err)
		}
	}()

	encryptedWriter := cipher.StreamWriter{S: stream, W: session_conn}

	buffer := make([]byte, MAX_TCP_BUFFER)

	if _, err := io.CopyBuffer(encryptedWriter, client_conn, buffer); err != nil {
		log.Printf("Failed forwarding to Cloud: %s\n", err)
	}

}

func (ac AppHead) dialCloud() *net.TCPConn {
	cloud_address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ac.CloudHost, ac.CloudPort))
	if err != nil {
		log.Fatalf("ERROR: Resolve TCP address %s:%d, %s \n", ac.CloudHost, ac.CloudPort, err)
		return nil
	}

	s_conn, err := net.DialTCP("tcp", nil, cloud_address)
	if err != nil {
		log.Printf("ERROR: Can't connect to Cloud %s, %s \n", cloud_address, err)
		return nil
	}

	return s_conn
}

func (ac AppHead) sendConnect(session_conn net.Conn) bool {
	var buffer bytes.Buffer

	buffer.WriteByte(APP_CONNECT)

	lb := uint16(len(ac.EdgeID))
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, lb)
	buffer.Write(length)

	buffer.WriteString(ac.EdgeID)

	_, err := session_conn.Write(buffer.Bytes())
	if err != nil {
		log.Printf("ERROR: Send Connect to cloud, %s\n", err)
		return false
	}

	return receiveACK(session_conn)
}
