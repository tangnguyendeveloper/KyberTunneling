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

type EdgeHead struct {
	CloudHost  string
	CloudPort  uint
	EdgeID     string
	cloud_conn *net.TCPConn

	DestinationServiceHost string
	DestinationServicePort uint
}

func NewEdgeHead(CloudHost string, CloudPort uint, DestinationServiceHost string, DestinationServicePort uint, EdgeID string) *EdgeHead {
	return &EdgeHead{
		CloudHost: CloudHost, CloudPort: CloudPort, EdgeID: EdgeID,
		DestinationServiceHost: DestinationServiceHost,
		DestinationServicePort: DestinationServicePort,
	}
}

func (ec *EdgeHead) Start() {

	cloud_address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ec.CloudHost, ec.CloudPort))
	if err != nil {
		log.Fatalf("ERROR: Resolve TCP address %s:%d, %s \n", ec.CloudHost, ec.CloudPort, err)
	}

	for {
		ec.cloud_conn, err = net.DialTCP("tcp", nil, cloud_address)
		if err != nil {
			log.Printf("ERROR: Can't connect to %s, %s \n", cloud_address, err)
		}

		ok := ec.sendConnect()
		if ok {
			break
		}
		time.Sleep(time.Second * 30)
	}

	go ec.trafficHandle()

	for {
		time.Sleep(time.Minute * 15)

		ec.cloud_conn, err = net.DialTCP("tcp", nil, cloud_address)
		if err != nil {
			log.Printf("ERROR: Can't connect to %s, %s \n", cloud_address, err)
		}

		ec.sendAlive()
	}

}

func (ec *EdgeHead) trafficHandle() {
	for {
		open_link_request, err := ReceiveCommandMessage(ec.cloud_conn)
		if err != io.EOF && err != nil {
			log.Printf("ERROR: ReceiveCommandMessage, %s\n", err)
			ec.reconnect()
			continue

		} else if err == io.EOF {
			time.Sleep(time.Millisecond)
			continue
		}

		if open_link_request.Command != EDGE_OPEN_LINK_REQUEST {
			continue
		}

		session_conn := ec.dialCloud()
		if session_conn == nil {
			continue
		}

		ok := sendOpenLinkResponse(session_conn, open_link_request)
		if !ok {
			continue
		}

		service_conn := ec.dialService()
		if service_conn == nil {
			continue
		}

		log.Printf("INFO: Start session %x\n", string(open_link_request.Parameter))
		go edgeForwarding(session_conn, service_conn)
	}
}

func (ec *EdgeHead) reconnect() {
	for {
		ec.cloud_conn = ec.dialCloud()
		if ec.cloud_conn == nil {
			time.Sleep(time.Second * 5)
			continue
		}
		ec.sendConnect()
		return
	}
}

// func edgeForwarding(session_conn net.Conn, service_conn net.Conn) {
// 	key := []byte("1234567890abohtdgetonahuytekhu@%")

// 	block, err := aes.NewCipher(key)
// 	if err != nil {
// 		log.Printf("ERROR: Failed to create AES cipher: %s\n", err)
// 		return
// 	}

// 	iv := make([]byte, aes.BlockSize)
// 	stream := cipher.NewCTR(block, iv)

// 	defer session_conn.Close()
// 	defer service_conn.Close()

// 	go func() {

// 		encryptedWriter := cipher.StreamWriter{S: stream, W: session_conn}

// 		buffer := make([]byte, MAX_TCP_BUFFER)

// 		if _, err := io.CopyBuffer(encryptedWriter, service_conn, buffer); err != nil {
// 			log.Printf("Failed forwarding to Cloud: %s\n", err)
// 		}
// 	}()

// 	decryptedReader := cipher.StreamReader{S: stream, R: session_conn}

// 	buffer := make([]byte, MAX_TCP_BUFFER)

// 	if _, err := io.CopyBuffer(service_conn, decryptedReader, buffer); err != nil {
// 		log.Printf("Failed forwarding to Service: %s\n", err)
// 	}

// }

func edgeForwarding(session_conn net.Conn, service_conn net.Conn) {
	key := []byte("1234567890abohtdgetonahuytekhu@%")

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("ERROR: Failed to create AES cipher: %s\n", err)
		return
	}

	iv := make([]byte, aes.BlockSize)
	stream := cipher.NewCTR(block, iv)

	defer session_conn.Close()
	defer service_conn.Close()

	go func() {
		encryptedWriter := &cipher.StreamWriter{S: stream, W: session_conn}

		buffer := make([]byte, MAX_TCP_BUFFER)

		for {
			n, err := service_conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("Failed forwarding to Cloud: %s\n", err)
				}
				break
			}

			_, err = encryptedWriter.Write(buffer[:n])
			if err != nil {
				log.Printf("Failed forwarding to Cloud: %s\n", err)
				break
			}
		}
	}()

	decryptedReader := &cipher.StreamReader{S: stream, R: session_conn}

	buffer := make([]byte, MAX_TCP_BUFFER)

	for {
		n, err := decryptedReader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Failed forwarding to Service: %s\n", err)
			}
			break
		}

		_, err = service_conn.Write(buffer[:n])
		if err != nil {
			log.Printf("Failed forwarding to Service: %s\n", err)
			break
		}
	}
}

func sendOpenLinkResponse(conn net.Conn, open_link_request *CommandMessage) bool {
	open_link_request.Command = EDGE_OPEN_LINK_RESPONSE

	_, err := conn.Write(open_link_request.ToByte())
	if err != nil {
		log.Printf("ERROR: sendOpenLinkResponse, %s\n", err)
		return false
	}

	return receiveACK(conn)
}

func (ec EdgeHead) dialCloud() *net.TCPConn {
	cloud_address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ec.CloudHost, ec.CloudPort))
	if err != nil {
		log.Printf("ERROR: Resolve TCP address %s:%d, %s \n", ec.CloudHost, ec.CloudPort, err)
		return nil
	}

	s_conn, err := net.DialTCP("tcp", nil, cloud_address)
	if err != nil {
		log.Printf("ERROR: Can't connect to Cloud %s, %s \n", cloud_address, err)
		return nil
	}

	return s_conn
}

func (ec EdgeHead) dialService() *net.TCPConn {
	service_address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ec.DestinationServiceHost, ec.DestinationServicePort))
	if err != nil {
		log.Printf("ERROR: Resolve TCP address %s:%d, %s \n", ec.DestinationServiceHost, ec.DestinationServicePort, err)
		return nil
	}

	s_conn, err := net.DialTCP("tcp", nil, service_address)
	if err != nil {
		log.Printf("ERROR: Can't connect to Service %s, %s \n", service_address, err)
		return nil
	}

	return s_conn
}

func receiveACK(conn net.Conn) bool {
	ack, err := ReceiveCommandMessage(conn)
	if err != nil {
		log.Printf("ERROR: Receive ACK from cloud, %s\n", err)
		return false
	}

	if ack.Command == ERROR_ACK {
		log.Printf("ERROR: ERROR_ACK, %s\n", string(ack.Parameter))
		return false
	}

	if ack.Command == SUCCESS_ACK {
		log.Printf("INFO: SUCCESS_ACK, %s\n", string(ack.Parameter))
		return true
	}
	return false
}

func (ec EdgeHead) sendConnect() bool {
	var buffer bytes.Buffer

	buffer.WriteByte(EDGE_CONNECT)

	lb := uint16(len(ec.EdgeID))
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, lb)
	buffer.Write(length)

	buffer.WriteString(ec.EdgeID)

	_, err := ec.cloud_conn.Write(buffer.Bytes())
	if err != nil {
		log.Printf("ERROR: Send Connect to cloud, %s\n", err)
		return false
	}

	return receiveACK(ec.cloud_conn)
}

func (ec EdgeHead) sendAlive() {
	var buffer bytes.Buffer

	buffer.WriteByte(EDGE_ALIVE)

	lb := uint16(len(ec.EdgeID))
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, lb)
	buffer.Write(length)

	buffer.WriteString(ec.EdgeID)

	_, err := ec.cloud_conn.Write(buffer.Bytes())
	if err != nil {
		log.Printf("ERROR: Send Alive to cloud, %s\n", err)
	}
}
