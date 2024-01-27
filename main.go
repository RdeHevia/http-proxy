package main

import (
	"log"
	"net"
)

/*
ALGO:
- Listen por incoming connections TCP
- If TCP connection: Start a go routine
	- -> Accept
	- read the content (multiple packets?)
	- open TCP connection with backend
	- send data
	-
------
CONSTANTS:
TCP_PORT=1234
ENDPOINT_HOST=localhost
ENPOINT_PORT=1234
*/

func proxyRequest(connection net.Conn) {
	requestData := make([]byte, 4000)
	bytesRequest, err := connection.Read(requestData)
	if err != nil {
		log.Println(err)
		return // TBD
	}

	log.Printf(" ? -> %v ; %vB", connection.RemoteAddr(), bytesRequest)

	connectionEndpoint, err := net.Dial("tcp", "localhost:9000")
	connectionEndpoint.Write(requestData)
	log.Printf(" %v -> %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesRequest)
	responseData := make([]byte, 4000)
	bytesResponse, err := connectionEndpoint.Read(responseData)
	if err != nil {
		log.Println(err)
		return // TBD (Close too?)
	}

	log.Printf(" %v <- %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesResponse)

	connection.Write(responseData)

	connection.Close()
}

func main() {
	tcpListener, err := net.Listen("tcp", "localhost:1234")
	if err != nil {
		log.Panicln(err)
	}

	log.Printf("The proxy is listening for connections in port %v", 1234)

	for {
		connection, err := tcpListener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go proxyRequest(connection)
	}

}
