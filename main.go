package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
)

const (
	PROXY_ENDPOINT  = ":1234"
	SERVER_ENDPOINT = "localhost:9000"
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

func forwardRequest(id string, clientConnection net.Conn, serverConnection net.Conn) {
	client := "client"
	proxyAddress := clientConnection.RemoteAddr()
	remoteServerAddress := serverConnection.RemoteAddr()

	requestData := make([]byte, 1500)
	bytesRequest, _ := clientConnection.Read(requestData)
	if bytesRequest == 0 {
		print(id, "Empty HTTP request received from the client")
		return
	}

	// printMessage(id, requestData)

	// Open connection with remote server and forward request

	print(id, "%v ==> %v ... %v ; %vB", client, proxyAddress, remoteServerAddress, bytesRequest)

	serverConnection.Write(requestData)
	print(id, "%v ... %v ==> %v ; %vB", client, proxyAddress, remoteServerAddress, bytesRequest)
}

func forwardResponse(id string, clientConnection net.Conn, serverConnection net.Conn) {
	client := "client"
	proxyAddress := clientConnection.RemoteAddr()
	remoteServerAddress := serverConnection.RemoteAddr()

	// Receive segments from remote server and forward them to client until done
	segmentCount := 0
	for {
		print(id, " - response segment count: %v", segmentCount)

		responseData := make([]byte, 1500)
		bytesResponse, _ := serverConnection.Read(responseData)

		print(id, "%v ... %v <== %v ; %vB", client, proxyAddress, serverConnection.LocalAddr(), bytesResponse)
		// printMessage(id, responseData)

		if bytesResponse == 0 {
			print(id, "Empty HTTP response received from the server")
			serverConnection.Close()
			print(id, "TCP connection with localhost:9000 closed")
			break
		}

		clientConnection.Write(responseData[:bytesResponse])
		print(id, "%v <== %v ... %v ; %vB", client, proxyAddress, remoteServerAddress, bytesResponse)

		segmentCount++
		// time.Sleep(1 * time.Second) // RdH todo delete
	}
}

func proxy(id string, clientConnection net.Conn) {
	// Open connection with remote server
	serverConnection, _ := net.Dial("tcp", SERVER_ENDPOINT)

	forwardRequest(id, clientConnection, serverConnection)
	forwardResponse(id, clientConnection, serverConnection)
}

func main() {
	tcpListener, err := net.Listen("tcp", PROXY_ENDPOINT)
	if err != nil {
		log.Panicln(err)
	}

	log.Printf("The proxy is listening for connections in %v", PROXY_ENDPOINT)

	for {
		connection, err := tcpListener.Accept()
		id, _ := GenerateRandomString(5)

		log.Printf("%v - TCP Connection accepted: %v", id, connection.RemoteAddr())
		if err != nil {
			log.Println(err)
			continue
		}

		go proxy(id, connection)
	}
}
