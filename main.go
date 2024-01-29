package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
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

func printMessage(id string, buffer []byte) {
	// message := strings.ReplaceAll(string(buffer), "\r", "")
	message := string(buffer)
	print(id, message)
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

func proxyRequest(id string, connection net.Conn) {
	packageCount := 0
	connectionEndpoint, _ := net.Dial("tcp", "localhost:9000")

	requestData := make([]byte, 1500) // RdH change
	bytesRequest, _ := connection.Read(requestData)
	if bytesRequest == 0 {
		// log.Println("RdH request bytes 0")
		print(id, "RdH request bytes 0")
		return
	}

	print(id, " %v -> %v ; %vB", "?", connection.RemoteAddr(), bytesRequest)
	printMessage(id, requestData)

	connectionEndpoint.Write(requestData)
	print(id, " %v -> %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesRequest)
	// time.Sleep(100 * time.Millisecond) // Rdh this fixes some kind of racing condition. Investigate
	for {
		print(id, "rdh iteration - %v", packageCount)

		responseData := make([]byte, 1500)
		bytesResponse, _ := connectionEndpoint.Read(responseData)

		print(id, " %v <- %v ; %vB", connection.RemoteAddr(), connectionEndpoint.RemoteAddr(), bytesResponse)
		printMessage(id, responseData)

		if bytesResponse == 0 {
			print(id, "rdh bytes 0")
			break
		}

		connection.Write(responseData[:bytesResponse])

		print(id, " %v <- %v ; %vB", "?", connection.RemoteAddr(), bytesResponse)

		packageCount++
		// time.Sleep(1 * time.Second) // RdH todo delete
	}

	print(id, "rdh here")
	connectionEndpoint.Close()
	print(id, "TCP connection with localhost:9000 closed")
	// connection.Close() // RdH: Probably don't need it (connection closed by the client)
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
		// proxyRequest(id, connection)
	}

}
