package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
)

type Request struct {
	id             string
	connectionId   string
	requestLine    string
	headers        map[string]string
	lengthBodyRead int
}

const (
	PROXY_ENDPOINT  = ":1234"
	SERVER_ENDPOINT = "localhost:9000"
)

func parseHTTP(id string, buffer []byte, bytesCount int) {
	text := string(buffer[:bytesCount])
	print(id, text)
}

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
	parseHTTP(id, requestData, bytesRequest)

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

/*
DATA STRUCTURE:

request {
	id string
	connectionId string
	requestLine: "GET / HTTP/1.1 200"
	headers: map
	contentLength int
	bytesReceivedAfterHeader int
}
ALGO exit loop:
- read response -> number of bytes
- try to parse headers
	- yes:
		- if requestLine exist -> initate request object
		-	record contentLength
		- if contentLength == 0 -> return
		- if contentLength > 0
	- no:
		- if bytes < contentLength:
			- bytesReceivedAfterHeader += bytes
		- if bytes >= contentLength:
			- close connection with server
----------------
- read response -> number of bytes
- done, err := processHTTPResponse
- if done -> exit loop
*/

func processHTTPResponseFromServer(id string, serverConnection net.Conn, client string, proxyAddress net.Addr) (responseData []byte, done bool, err error) {
	responseData = make([]byte, 1500)
	bytesResponse, err := serverConnection.Read(responseData)
	parseHTTP(id, responseData, bytesResponse)

	print(id, "%v ... %v <== %v ; %vB", client, proxyAddress, serverConnection.LocalAddr(), bytesResponse)

	if err != nil && err.Error() != "EOF" {
		print(id, err.Error())
		return nil, false, err
	}

	// printMessage(id, responseData)

	if bytesResponse == 0 {
		return responseData[:bytesResponse], true, nil
	}

	return responseData[:bytesResponse], false, nil
}

func forwardResponse(id string, clientConnection net.Conn, serverConnection net.Conn) error {
	client := "client"
	proxyAddress := clientConnection.RemoteAddr()
	remoteServerAddress := serverConnection.RemoteAddr()

	// Receive segments from remote server and forward them to client until done
	segmentCount := 0
	for {
		print(id, " - response segment count: %v", segmentCount)

		responseData, done, err := processHTTPResponseFromServer(id, serverConnection, client, proxyAddress)

		if err != nil {
			print(id, err.Error())
			serverConnection.Close()
			clientConnection.Close()
			return err
		}

		bytesResponse, err := clientConnection.Write(responseData)
		print(id, "%v <== %v ... %v ; %vB", client, proxyAddress, remoteServerAddress, bytesResponse)

		if err != nil && err.Error() != "EOF" {
			print(id, err.Error())
			return err
		}

		if done {
			print(id, "Empty HTTP response received from the server")
			serverConnection.Close()
			print(id, "TCP connection with localhost:9000 closed")
			return nil
		}

		segmentCount++
		// time.Sleep(1 * time.Second) // RdH todo delete
	}
}

func proxy(id string, clientConnection net.Conn) error {
	// Open connection with remote server
	serverConnection, err := net.Dial("tcp", SERVER_ENDPOINT)
	if err != nil {
		log.Println(err)
		clientConnection.Close()
		return err
	}

	forwardRequest(id, clientConnection, serverConnection)
	print(id, "rdh after forward request")
	forwardResponse(id, clientConnection, serverConnection)
	print(id, "rdh after forward response")

	return nil
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

		// go proxy(id, connection)
		proxy(id, connection)
		print(id, "rdh after proxy")
	}
}
