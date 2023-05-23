package TCPlib

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
)

// Command + length + Parameter

type CommandMessage struct {
	Command   uint8
	Parameter []byte
}

func ReceiveCommandMessage(conn net.Conn) (*CommandMessage, error) {
	reader := bufio.NewReader(conn)

	command, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	lb := make([]byte, 2)
	_, err = reader.Read(lb)
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint16(lb)
	parameter := make([]byte, length)
	_, err = reader.Read(parameter)
	if err != nil {
		return nil, err
	}

	return &CommandMessage{Command: command, Parameter: parameter}, nil
}

func (cmd_msg CommandMessage) ToByte() []byte {
	var buffer bytes.Buffer

	buffer.WriteByte(cmd_msg.Command)

	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(cmd_msg.Parameter)))
	buffer.Write(length)
	buffer.Write(cmd_msg.Parameter)

	return buffer.Bytes()
}
