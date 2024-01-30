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

func forwardRequest(id string, clientConnection net.Conn, serverConnection net.Conn) error {
	client := "client"
	proxyAddress := clientConnection.RemoteAddr()
	remoteServerAddress := serverConnection.RemoteAddr()

	requestData := make([]byte, 1500)
	bytesRequest, err := clientConnection.Read(requestData)
	if err != nil && err.Error() != "EOF" {
		print(id, err.Error())
		return err
	}

	if bytesRequest == 0 {
		print(id, "Empty HTTP request received from the client")
		return nil
	}

	// printMessage(id, requestData)

	// Open connection with remote server and forward request

	print(id, "%v ==> %v ... %v ; %vB", client, proxyAddress, remoteServerAddress, bytesRequest)

	_, err = serverConnection.Write(requestData[:bytesRequest])
	print(id, "%v ... %v ==> %v ; %vB", client, proxyAddress, remoteServerAddress, bytesRequest)

	return err
}

func forwardResponse(id string, clientConnection net.Conn, serverConnection net.Conn) error {
	client := "client"
	proxyAddress := clientConnection.RemoteAddr()
	remoteServerAddress := serverConnection.RemoteAddr()

	// Receive segments from remote server and forward them to client until done
	segmentCount := 0
	for {
		print(id, " - response segment count: %v", segmentCount)

		responseData := make([]byte, 1500)
		bytesResponse, err := serverConnection.Read(responseData)

		print(id, "%v ... %v <== %v ; %vB", client, proxyAddress, serverConnection.LocalAddr(), bytesResponse)

		if err != nil && err.Error() != "EOF" {
			print(id, err.Error())
			return err
		}

		// printMessage(id, responseData)

		if err != nil && err.Error() != "EOF" {
			print(id, err.Error())
			return err
		}

		if bytesResponse == 0 {
			print(id, "Empty HTTP response received from the server")
			serverConnection.Close()
			print(id, "TCP connection with localhost:9000 closed")
			return nil
		}

		_, err = clientConnection.Write(responseData[:bytesResponse])
		print(id, "%v <== %v ... %v ; %vB", client, proxyAddress, remoteServerAddress, bytesResponse)

		if err != nil && err.Error() != "EOF" {
			print(id, err.Error())
			return err
		}

		segmentCount++
		// time.Sleep(1 * time.Second) // RdH todo delete
	}

	return nil
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
