package main

import (
	"log"
	"net"
	"time"
)

func CreateUDPConnect(connectAddr string, listenPort string) (*net.UDPConn, error) {
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

func main() {
	for {
		// 连接服务端
		conn, err := CreateUDPConnect("127.0.0.1", "8083")
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Client started on %s\n", conn.LocalAddr())

		// 发送 PING
		_, err = conn.Write([]byte("8083"))
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Sent: PING")

		// 接收 PONG
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(30 * time.Second)) // 设置超时

		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Failed to receive PONG: %s", err.Error())
			continue
		}

		log.Printf("Received: %s %d", string(buf[:n]), n)
		//time.Sleep(1 * time.Second)
		//time.Sleep(100 * time.Millisecond)
		time.Sleep(1 * time.Millisecond)
		go func() {
			defer conn.Close()
		}()
	}
}
