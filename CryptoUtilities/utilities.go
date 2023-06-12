package CryptoUtilities

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httputil"
)

func ReadHTTPRequest(conn net.Conn) ([]byte, error) {

	// Read the incoming request using bufio
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		return nil, err
	}

	// Dump the request to a byte array
	dump, err := httputil.DumpRequest(request, true)
	if err != nil {
		return nil, err
	}

	return dump, nil
}

func ReadHTTPResponse(conn net.Conn) ([]byte, error) {

	// Read the incoming request using bufio
	reader := bufio.NewReader(conn)
	request, err := http.ReadResponse(reader, nil)
	if err != nil {
		return nil, err
	}

	// Dump the request to a byte array
	dump, err := httputil.DumpResponse(request, true)
	if err != nil {
		return nil, err
	}

	return dump, nil
}
