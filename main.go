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
	requestData := make([]byte, 40000) // RdH change
	bytesRequest, err := connection.Read(requestData)
	if bytesRequest == 0 {
		log.Println("RdH request bytes 0")
		return
	}
	if err != nil {
		log.Println(err)
		return // TBD
	}

	log.Printf(" %v -> %v ; %vB", connection.LocalAddr(), connection.RemoteAddr(), bytesRequest)

	connectionEndpoint, err := net.Dial("tcp", "localhost:9000")
	connectionEndpoint.Write(requestData)
	log.Printf(" %v -> %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesRequest)
	responseData := make([]byte, 40000)
	bytesResponse, err := connectionEndpoint.Read(responseData)
	if err != nil {
		log.Println(err)
		return // TBD (Close too?)
	}

	log.Printf(" %v <- %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesResponse)

	connection.Write(responseData)
	log.Printf(" %v <- %v ; %vB", connection.LocalAddr(), connection.RemoteAddr(), bytesResponse)

	packageCount := 1
	for {
		log.Println("rdh iteration", packageCount)
		bytesResponse, err = connectionEndpoint.Read(responseData)
		log.Printf(" %v <- %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesResponse)

		// if err != nil {
		// 	log.Println(err)
		// 	log.Println("rdh err!", err)
		// 	return // TBD (Close too?)
		// }
		connection.Write(responseData)
		log.Printf(" %v <- %v ; %vB", connection.LocalAddr(), connection.RemoteAddr(), bytesResponse)

		if bytesResponse == 0 {
			log.Println("rdh bytes 0")
			break
		}

		packageCount++
	}
	log.Println("rdh here")
	connectionEndpoint.Close()
	log.Println("TCP connection with localhost:9000 closed")
	connection.Close()
	log.Printf("TCP connection with %v closed", connection.LocalAddr())
}

func main() {
	tcpListener, err := net.Listen("tcp", "localhost:1234")
	if err != nil {
		log.Panicln(err)
	}

	log.Printf("The proxy is listening for connections in port %v", 1234)

	for {
		connection, err := tcpListener.Accept()

		log.Println("TCP Connection accepted", connection.LocalAddr())
		if err != nil {
			log.Println(err)
			continue
		}

		go proxyRequest(connection)
	}

}
