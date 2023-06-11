package TCPlib

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/tangnguyendeveloper/KyberTunneling/CryptoUtilities"
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

	defer session_conn.Close()
	defer client_conn.Close()

	key := []byte("12345678900987654321asdfghjklpoi")

	// go io.Copy(session_conn, client_conn)
	// io.Copy(client_conn, session_conn)

	go func() {
		length := make([]byte, 2)
		for {
			n, err := session_conn.Read(length)
			if err != nil {
				log.Println(err)
				break
			}
			if n != 2 {
				continue
			}

			lb := binary.BigEndian.Uint16(length)
			ciphertext := make([]byte, lb)
			n, err = session_conn.Read(ciphertext)
			if err != nil {
				log.Println(err)
				break
			}
			if uint16(n) != lb {
				continue
			}

			plaintext, err := CryptoUtilities.Decrypt(key, ciphertext)
			if err != nil {
				log.Println(err)
				break
			}

			_, err = client_conn.Write(plaintext)
			if err != nil {
				log.Println(err)
				break
			}

		}
	}()

	plaintext := make([]byte, MAX_TCP_BUFFER)
	for {
		n, err := client_conn.Read(plaintext)
		if err != nil {
			log.Println(err)
			break
		}
		if n < 0 {
			continue
		}

		ciphertext, err := CryptoUtilities.Encrypt(key, plaintext[:n])
		if err != nil {
			log.Println(err)
			break
		}

		length := make([]byte, 2)
		binary.BigEndian.PutUint16(length, uint16(len(ciphertext)))

		_, err = session_conn.Write(length)
		if err != nil {
			log.Println(err)
			break
		}
		_, err = session_conn.Write(ciphertext)
		if err != nil {
			log.Println(err)
			break
		}

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
