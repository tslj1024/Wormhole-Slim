package main

import (
	"log"
	"net"
)

func CreateUDPListen(listenAddr string, listenPort string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr+":"+listenPort)
	if err != nil {
		return nil, err
	}
	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return udpListener, err
}

func main() {
	conn, err := CreateUDPListen("", "8083")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	log.Printf("Server listening on %s\n", ":8083")

	buf := make([]byte, 1400)
	for {
		n, clientAddr, _ := conn.ReadFromUDP(buf)

		msg := string(buf[:n])
		log.Printf("Received from %s: \"%s\" %d", clientAddr.String(), msg, n)

		log.Printf("1")
		_, err = conn.WriteToUDP([]byte("PONG-"+msg), clientAddr)
		log.Printf("2")
		if err != nil {
			log.Printf("Failed to send PONG: %v", err)
		}
	}
}
