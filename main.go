package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"
)

func GenerateRandomString(length int) (string, error) {
	str := make([]byte, length/2)
	_, err := rand.Read(str)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(str), nil
}

func print(id string, messageRaw string, v ...any) {
	message := fmt.Sprintf(messageRaw, v...)
	log.Printf("%v - %v", id, message)
}

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

// func proxyRequest(connection net.Conn) {
// 	connectionEndpoint, _ := net.Dial("tcp", "localhost:9000")

// 	count := 0
// 	requestData := make([]byte, 1500) // RdH change
// 	for {
// 		log.Println("rdh request forward count", count)
// 		log.Println("rdh here1")
// 		bytesRequest, _ := connection.Read(requestData)
// 		log.Println("rdh here2")

// 		// if err != nil {
// 		// 	log.Println(err)
// 		// 	break // TBD
// 		// }
// 		log.Printf(" %v -> %v ; %vB", connection.LocalAddr(), connection.RemoteAddr(), bytesRequest)

// 		connectionEndpoint.Write(requestData)
// 		log.Printf(" %v -> %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesRequest)

// 		if bytesRequest == 0 {
// 			log.Println("RdH request bytes 0")
// 			break
// 		}
// 		count++
// 	}

// 	for {
// 		responseData := make([]byte, 1500)
// 		bytesResponse, _ := connectionEndpoint.Read(responseData)
// 		log.Printf(" %v <- %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesResponse)
// 		connection.Write(responseData)

// 		log.Printf(" %v <- %v ; %vB", connection.LocalAddr(), connection.RemoteAddr(), bytesResponse)

// 		if bytesResponse == 0 {
// 			log.Println("rdh bytes 0")
// 			break
// 		}
// 	}

// 	connectionEndpoint.Close()
// }

func proxyRequest(id string, connection net.Conn) {
	requestData := make([]byte, 1500) // RdH change
	bytesRequest, err := connection.Read(requestData)
	if bytesRequest == 0 {
		// log.Println("RdH request bytes 0")
		print(id, "RdH request bytes 0")
		return
	}
	if err != nil {
		log.Println(err)
		return // TBD
	}

	print(id, " %v -> %v ; %vB", connection.LocalAddr(), connection.RemoteAddr(), bytesRequest)

	connectionEndpoint, err := net.Dial("tcp", "localhost:9000")
	connectionEndpoint.Write(requestData)

	print(id, " %v -> %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesRequest)

	responseData := make([]byte, 1500)
	bytesResponse, err := connectionEndpoint.Read(responseData)
	if err != nil {
		log.Println(err)
		return // TBD (Close too?)
	}

	print(id, " %v <- %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesResponse)

	connection.Write(responseData)

	print(id, " %v <- %v ; %vB", connection.LocalAddr(), connection.RemoteAddr(), bytesResponse)

	packageCount := 1
	for {
		print(id, "rdh iteration - %v", packageCount)
		bytesResponse, err = connectionEndpoint.Read(responseData)
		print(id, " %v <- %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesResponse)

		// if err != nil {
		// 	log.Println(err)
		// 	log.Println("rdh err!", err)
		// 	return // TBD (Close too?)
		// }
		connection.Write(responseData)
		print(id, " %v <- %v ; %vB", connection.LocalAddr(), connection.RemoteAddr(), bytesResponse)

		if bytesResponse == 0 {
			print(id, "rdh bytes 0")
			break
		}

		packageCount++
		time.Sleep(1 * time.Second) // RdH todo delete
	}

	print(id, "rdh here")
	connectionEndpoint.Close()
	print(id, "TCP connection with localhost:9000 closed")
	// connection.Close() // RdH: Probably don't need it (connection closed by the client)
	// log.Printf("TCP connection with %v closed", connection.LocalAddr())
}

func main() {
	tcpListener, err := net.Listen("tcp", "localhost:1234")
	if err != nil {
		log.Panicln(err)
	}

	log.Printf("The proxy is listening for connections in port %v", 1234)

	for {
		connection, err := tcpListener.Accept()

		id, _ := GenerateRandomString(5)

		log.Printf("%v - TCP Connection accepted: %v", id, connection.LocalAddr())
		if err != nil {
			log.Println(err)
			continue
		}

		go proxyRequest(id, connection)
	}

}
