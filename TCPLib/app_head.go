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

	// key := []byte("1234567890abohtdgetonahuytekhu@%")

	// block, err := aes.NewCipher(key)
	// if err != nil {
	// 	log.Printf("ERROR: Failed to create AES cipher: %s\n", err)
	// 	return
	// }

	// iv := make([]byte, aes.BlockSize)
	// stream := cipher.NewCTR(block, iv)

	// defer session_conn.Close()
	// defer client_conn.Close()

	// go func() {

	// 	decryptedReader := cipher.StreamReader{S: stream, R: session_conn}

	// 	if _, err := io.Copy(client_conn, decryptedReader); err != nil {
	// 		log.Printf("Failed forwarding to App: %s\n", err)
	// 	}
	// }()

	// encryptedWriter := cipher.StreamWriter{S: stream, W: session_conn}

	// if _, err := io.Copy(encryptedWriter, client_conn); err != nil {
	// 	log.Printf("Failed forwarding to Cloud: %s\n", err)
	// }

	key := []byte("1234567890abohtdgetonahuytekhu@%")
	var done bool = false

	go func() {
		client_buffer := make([]byte, 4096)
		length := make([]byte, 2)

		for {
			err := client_conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			if err != nil {
				log.Printf("ERROR: set receive timeout client_conn, %s\n", err)
				break
			}
			n, err := client_conn.Read(client_buffer)
			if err != nil {
				log.Printf("ERROR: Receive from App, %s\n", err)
				break
			}

			if n < 1 {
				continue
			}

			ciphertext, err := CryptoUtilities.Encrypt(key, client_buffer[:n])
			if err != nil {
				log.Printf("ERROR: AES Encrypt, %s\n", err)
				break
			}

			n = len(ciphertext)
			binary.BigEndian.PutUint16(length, uint16(n))

			err = session_conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
			if err != nil {
				log.Printf("ERROR: set send timeout session_conn, %s\n", err)
				break
			}

			_, err = session_conn.Write(append(length, ciphertext...))
			if err != nil {
				log.Printf("ERROR: Forward to Cloud, %s\n", err)
				break
			}

		}

		client_conn.Close()
		session_conn.Close()

		done = true

	}()

	length1 := make([]byte, 2)

	for {
		err := session_conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		if err != nil {
			log.Printf("ERROR: set receive timeout session_conn, %s\n", err)
			break
		}

		n, _ := session_conn.Read(length1)
		if n != 2 {
			continue
		}

		lb := binary.BigEndian.Uint16(length1)
		session_buffer := make([]byte, lb)

		n, err = session_conn.Read(session_buffer)
		if err != nil {
			log.Printf("ERROR: Receive from Cloud, %s\n", err)
			break
		}

		if n != int(lb) {
			continue
		}

		plaintext, err := CryptoUtilities.Decrypt(key, session_buffer)
		if err != nil {
			log.Printf("ERROR: AES Decrypt, %s\n", err)
			break
		}

		err = client_conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
		if err != nil {
			log.Printf("ERROR: set send timeout client_conn, %s\n", err)
			break
		}

		_, err = client_conn.Write(plaintext)
		if err != nil {
			log.Printf("ERROR: Forward to App, %s\n", err)
			break
		}

	}

	if done {
		return
	}

	client_conn.Close()
	session_conn.Close()

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
