package util

import (
	"crypto/rand"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"net"
	"os"
)

const (
	CONNECT   = iota // connect
	HEARTBEAT        // heartbeat
	S_TO_C           // The server forwards user data to the client
	C_TO_S           // The client forwards the data returned by the target service to the server
)

// LoadConfig loads the configuration from the given path
func LoadConfig[T any](path string) *T {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading config file: %v", err)
		panic(err)
	}
	var config T
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Printf("Error parsing config file: %v", err)
		panic(err)
	}

	return &config
}

// CreateTCPListen Listen the connections from clients and users.
func CreateTCPListen(network string, listenAddr string, listenPort string) (*net.TCPListener, error) {
	tcpAddr, err := net.ResolveTCPAddr(network, listenAddr+":"+listenPort)
	if err != nil {
		return nil, err
	}
	tcpListener, err := net.ListenTCP(network, tcpAddr)
	if err != nil {
		return nil, err
	}
	return tcpListener, err
}

// CreateTCPConnect The client connects to the server.
func CreateTCPConnect(network string, connectAddr string, listenPort string) (*net.TCPConn, error) {
	tcpAddr, err := net.ResolveTCPAddr(network, connectAddr+":"+listenPort)
	if err != nil {
		return nil, err
	}
	tcpConn, err := net.DialTCP(network, nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	return tcpConn, err
}

func CreateUDPListen(network string, listenAddr string, listenPort string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr(network, listenAddr+":"+listenPort)
	if err != nil {
		return nil, err
	}
	udpListener, err := net.ListenUDP(network, udpAddr)
	if err != nil {
		return nil, err
	}
	return udpListener, err
}

func CreateUDPConnect(network string, connectAddr string, listenPort string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", connectAddr+":"+listenPort)
	if err != nil {
		return nil, err
	}
	tcpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}
	return tcpConn, err
}

// GetDataFromConnection Getting data from the connection
func GetDataFromConnection(bufSize int, conn *net.TCPConn) ([]byte, error) {
	b := make([]byte, 0)
	for {
		data := make([]byte, bufSize)
		n, err := conn.Read(data)
		if err != nil {
			return nil, err
		}
		b = append(b, data[:n]...)
		if n < bufSize {
			break
		}
	}
	return b, nil
}

// GenerateUUID create SessionID. The SessionID is used to address the issue of handling multiple connections on a single port.
func GenerateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
