package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

type MessageTracker struct {
	Id             string
	ConnectionId   string
	RequestLine    string
	Headers        map[string]string
	LengthBodyRead int
}

const (
	PROXY_ENDPOINT  = ":1234"
	SERVER_ENDPOINT = "localhost:9000"
)

// func logStruct(v any) {
// 	json, _ := json.MarshalIndent(v, "", "  ")
// 	log.Printf("%+v\n", string(json))
// }

// func parseHTTP(id string, buffer []byte, bytesCount int) { // RdH: TODO refactor
// 	text := string(buffer[:bytesCount])
// 	print(id, text)
// }

func print(connectionId string, messageRaw string, v ...any) {
	message := fmt.Sprintf(messageRaw, v...)
	log.Printf("%v - %v", connectionId, message)
}

func printMessage(connectionId string, buffer []byte) {
	message := string(buffer)
	print(connectionId, message)
}

func printStruct(connectionId string, v any) {
	json, _ := json.MarshalIndent(v, "", "  ")
	log.Printf("%v: %+v\n", connectionId, string(json))
}

/*
Assume message is either:
- Headers + body1; body2... bodyN
- all Headers; body1, body2... bodyN
-------
ALGO:
Split "\r\n\r\n"
- if len == 0 -> empty message -> return
- if len == 1 && ! tracker.Headers -> it's the Headers
	- parseHeaders and add them to the tracker
- if len ==1 && tracker.Headers -> it's the body
	- tracker.readBody += byteCount
- if len == 2 -> it's the Headers and body1
	- parseHeaders and add them to the tracker
	- tracker.readBody += byteCount


*/

func parseHeaders(headersRaw string, tracker *MessageTracker) {
	for _, line := range strings.Split(headersRaw, "\n") {
		keyAndVal := strings.Split(strings.TrimSpace(line), ": ")
		if len(keyAndVal) == 0 {
			return
		}

		if len(keyAndVal) == 1 {
			tracker.RequestLine = keyAndVal[0]
		} else {
			headerName := keyAndVal[0]
			headerValue := keyAndVal[1]
			tracker.Headers[headerName] = headerValue
		}
	}
}

func parseMessage(data []byte, bytesCount int, tracker *MessageTracker) (err error) {
	text := string(data[:bytesCount])
	print(tracker.ConnectionId, "rdh parseMessage:")
	// print(tracker.connectionId, text)
	sections := strings.Split(text, "\r\n\r\n")
	log.Println("rdh sections length", len(sections))
	if len(sections) == 0 {
		return nil
	} else if len(sections) == 1 && len(tracker.Headers) == 0 {
		parseHeaders(sections[0], tracker)
	} else if len(sections) == 1 && len(tracker.Headers) > 0 {
		body := sections[0]
		tracker.LengthBodyRead += len([]byte(body))
	} else {
		parseHeaders(sections[0], tracker)
		tracker.LengthBodyRead += len([]byte(sections[1]))
	}

	return nil
}

func GenerateRandomString(length int) (string, error) {
	str := make([]byte, length/2)
	_, err := rand.Read(str)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(str), nil
}

func forwardRequest(connectionId string, clientConnection net.Conn, upstreamConnection net.Conn) (empty bool, err error) {
	client := "client"
	proxyAddress := clientConnection.RemoteAddr()
	remoteServerAddress := upstreamConnection.RemoteAddr()

	requestData := make([]byte, 1500)
	requestByteCount, err := clientConnection.Read(requestData)
	// parseHTTP(connectionId, requestData, requestByteCount)
	printMessage(connectionId, requestData)

	if err != nil && err.Error() != "EOF" {
		print(connectionId, err.Error())
		return false, err
	}

	if requestByteCount == 0 {
		print(connectionId, "Empty HTTP request received from the client")
		return true, nil
	}

	// printMessage(connectionId, requestData)

	// Open connection with remote server and forward request

	print(connectionId, "%v ==> %v ... %v ; %vB", client, proxyAddress, remoteServerAddress, requestByteCount)

	_, err = upstreamConnection.Write(requestData[:requestByteCount])
	print(connectionId, "%v ... %v ==> %v ; %vB", client, proxyAddress, remoteServerAddress, requestByteCount)

	return false, err
}

func processResponseFromUpstream(responseTracker *MessageTracker, upstreamConnection net.Conn, client string, proxyAddress net.Addr) (data []byte, done bool, err error) {
	data = make([]byte, 1500)
	byteCount, err := upstreamConnection.Read(data)

	if err != nil {
		return nil, false, err
	}

	print(responseTracker.ConnectionId, "%v ... %v <== %v ; %vB", client, proxyAddress, upstreamConnection.LocalAddr(), byteCount)

	err = parseMessage(data, byteCount, responseTracker)

	if err != nil {
		return data[:byteCount], false, err
	}

	contentLengthStr, ok := responseTracker.Headers["Content-Length"]
	if !ok {
		return data[:byteCount], false, nil
	}

	contentLength, err := strconv.Atoi(contentLengthStr)

	if err != nil {
		return data[:byteCount], false, err
	}

	if responseTracker.LengthBodyRead < contentLength {
		return data[:byteCount], false, nil
	}

	return data[:byteCount], true, nil
}

func forwardResponse(connectionId string, clientConnection net.Conn, upstreamConnection net.Conn) error {
	client := "client"
	proxyAddress := clientConnection.RemoteAddr()
	remoteServerAddress := upstreamConnection.RemoteAddr()

	responseId, err := GenerateRandomString(5)
	if err != nil {
		return err
	}

	responseTracker := MessageTracker{
		Id:             responseId,
		ConnectionId:   connectionId,
		RequestLine:    "",
		Headers:        map[string]string{},
		LengthBodyRead: 0,
	}

	// Receive segments from remote server and forward them to client until done
	segmentCount := 0
	for {
		print(connectionId, " - response segment count: %v", segmentCount)

		responseData, done, err := processResponseFromUpstream(&responseTracker, upstreamConnection, client, proxyAddress)

		printStruct(connectionId, responseTracker)

		if err != nil {
			print(connectionId, err.Error())
			upstreamConnection.Close()
			clientConnection.Close()
			return err
		}

		responseByteCount, err := clientConnection.Write(responseData)
		print(connectionId, "%v <== %v ... %v ; %vB", client, proxyAddress, remoteServerAddress, responseByteCount)

		if err != nil && err.Error() != "EOF" {
			print(connectionId, err.Error())
			return err
		}

		if done {
			print(connectionId, "The request %v has been completely received", responseTracker.Id)
			upstreamConnection.Close()
			print(connectionId, "TCP connection with upstream server closed")
			return nil
		}

		segmentCount++
		// time.Sleep(1 * time.Second) // RdH todo delete
	}
}

func proxySS(connectionId string, clientConnection net.Conn) error {
	// Open connection with remote server
	upstreamConnection, err := net.Dial("tcp", SERVER_ENDPOINT)
	if err != nil {
		log.Println(err)
		clientConnection.Close()
		return err
	}

	forwardRequest(connectionId, clientConnection, upstreamConnection)
	print(connectionId, "rdh after forward request")
	forwardResponse(connectionId, clientConnection, upstreamConnection)
	print(connectionId, "rdh after forward response")

	return nil
}

/*
Proxy algo:
Loop:
- open connection with upstream server
- empty := forwardRequest
- if empty -> break loop
- if not empty -> continue
*/

func proxy(connectionId string, clientConnection net.Conn) error {
	// Open connection with remote server

	for {
		upstreamConnection, err := net.Dial("tcp", SERVER_ENDPOINT)
		print(connectionId, "Connection with upstream server opened")
		if err != nil {
			log.Println(err)
			clientConnection.Close()
			return err
		}

		empty, _ := forwardRequest(connectionId, clientConnection, upstreamConnection)
		if empty {
			upstreamConnection.Close()
			return nil
		}

		forwardResponse(connectionId, clientConnection, upstreamConnection)
		// upstreamConnection.Close() // RdH refactor so it doesn't close the connection inside forwardResponse. it should only be closed in here
	}

	// return nil
}

func main() {
	tcpListener, err := net.Listen("tcp", PROXY_ENDPOINT)
	if err != nil {
		log.Panicln(err)
	}

	log.Printf("The proxy is listening for connections in %v", PROXY_ENDPOINT)

	for {
		connection, err := tcpListener.Accept()
		connectionId, _ := GenerateRandomString(5)

		log.Printf("%v - TCP Connection accepted: %v", connectionId, connection.RemoteAddr())
		if err != nil {
			log.Println(err)
			continue
		}

		go proxy(connectionId, connection)
	}
}
