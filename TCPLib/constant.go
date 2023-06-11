package TCPlib

const (
	//MAX_TCP_BUFFER = 65535

	EDGE_ALIVE   = byte(0)
	EDGE_CONNECT = byte(1)
	APP_CONNECT  = byte(2)
	SUCCESS_ACK  = byte(3)
	ERROR_ACK    = byte(4)

	EDGE_OPEN_LINK_REQUEST  = byte(5)
	EDGE_OPEN_LINK_RESPONSE = byte(6)
)
