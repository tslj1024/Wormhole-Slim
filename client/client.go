package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
	"util"
)

// Define configuration
type Config struct {
	Client struct {
		Host          string        `yaml:"host"`
		Port          string        `yaml:"port"`
		ClientId      string        `yaml:"clientId"`
		PingMaxCnt    int           `yaml:"pingMaxCnt"`
		PingInterval  time.Duration `yaml:"pingInterval"`
		ReconWaitTime time.Duration `yaml:"reconWaitTime"`
		BufSize       int           `yaml:"bufSize"`
		PackSize      int           `yaml:"packSize"`
		HoleInterval  time.Duration `yaml:"holeInterval"`
	} `yaml:"client"`
}

// wg Used to wait for all coroutines to finish.
var wg sync.WaitGroup

// Configuration
var cfg *Config

var udpConn *net.UDPConn

func ping(conn *net.TCPConn) {
	flag := false // is first connect
	cnt := 0      // ping failure count
	for {
		if flag {
			_, err := conn.Write([]byte{util.HEARTBEAT})
			if err != nil {
				cnt++
				if cnt >= cfg.Client.PingMaxCnt {
					log.Printf("Disconnected from the server: %s", err)
					return
				}
			}
			cnt = 0
		} else {
			data := make([]byte, len(cfg.Client.ClientId)+1)
			data[0] = util.CONNECT
			copy(data[1:], cfg.Client.ClientId)
			_, err := conn.Write(data)
			if err != nil {
				log.Printf("Registration Error: %s", err)
				return
			}
			flag = true
		}
		time.Sleep(cfg.Client.PingInterval)
	}
}

// messageForward Connect the data tunnel for data forwarding.
func messageForward(sessionId string, tHost string, tPort string) {
	dataConn, err := util.CreateTCPConnect("tcp", cfg.Client.Host, cfg.Client.Port)
	if err != nil {
		return
	}
	targetConn, err := util.CreateTCPConnect("tcp", tHost, tPort)
	if err != nil {
		return
	}
	data := make([]byte, 37)
	data[0] = util.C_TO_S
	copy(data[1:], sessionId)
	_, err1 := dataConn.Write(data)
	if err1 != nil {
		log.Printf("Data tunnel connection Error %s", err1)
		return
	}
	go func() {
		n, _ := io.Copy(targetConn, dataConn)
		log.Printf("[%s] S -> T len= %d B", sessionId, n)
	}()
	go func() {
		n, _ := io.Copy(dataConn, targetConn)
		log.Printf("[%s] T -> S len= %d B", sessionId, n)
	}()
}

func TCPClient() {

	for {
		conn, err := util.CreateTCPConnect("tcp", cfg.Client.Host, cfg.Client.Port)
		if err != nil {
			log.Printf(fmt.Sprintf("Error connecting to server: %s", err))
			time.Sleep(cfg.Client.ReconWaitTime)
			continue
		}
		log.Printf("Sever connecting SUCCESS：%s\n", conn.RemoteAddr().String())

		go ping(conn)

		for {
			data, err1 := util.GetDataFromConnection(cfg.Client.BufSize, conn)
			if err1 != nil {
				log.Printf("Disconnected from the server: %s", err1)
				break
			}

			n := len(data)
			for i := 0; i < n; {
				flag := data[i]
				i++
				if flag == util.S_TO_C {
					sessionId := string(data[i : i+36])
					tHostLen := int(data[i+36])
					tHost := string(data[i+37 : i+37+tHostLen])
					tPortLen := int(data[i+37+tHostLen])
					tPort := string(data[i+38+tHostLen : i+38+tHostLen+tPortLen])
					i += 38 + tHostLen + tPortLen
					go messageForward(sessionId, tHost, tPort)
				}
			}
		}
		time.Sleep(cfg.Client.ReconWaitTime)
	}
}

func UDPClient() {
	var err error
	for {
		udpConn, err = util.CreateUDPConnect("udp", cfg.Client.Host, cfg.Client.Port)
		if err != nil {
			log.Printf("UDPClient connect Error: %s\n", err.Error())
			time.Sleep(cfg.Client.HoleInterval)
			continue
		}
		log.Printf("UDPClient connect success: %s", udpConn.RemoteAddr().String())
		data := make([]byte, cfg.Client.PackSize)
		clientIdLen := len(cfg.Client.ClientId)
		data[0] = util.CONNECT
		copy(data[1:], cfg.Client.ClientId)
		_, _ = udpConn.Write(data[:clientIdLen+1])
		//go ping(udpConn) // You shouldn't need to ping, just reconnect every once in a while
		go handleConn(udpConn)
		// Every once in a while, holes are drilled into the server
		time.Sleep(cfg.Client.HoleInterval)
	}
}

func handleConn(conn *net.UDPConn) {
	buf := make([]byte, cfg.Client.PackSize)
	var n int
	var length int
	for {
		// Accept the S data first, then send it to T, accept the return of T, and then send it to S
		n, _, _ = conn.ReadFromUDP(buf)
		if n == 0 {
			continue
		}
		sessionId := string(buf[1:37])
		thostLen := buf[37]
		thost := string(buf[38 : 38+thostLen])
		tportLen := buf[38+thostLen]
		tport := string(buf[39+thostLen : 39+thostLen+tportLen])
		data := buf[39+thostLen+tportLen : n]

		tConn, _ := util.CreateUDPConnect("udp", thost, tport)
		length, _ = tConn.Write(data)
		log.Printf("[%s] S -> T len=%d", sessionId, length)
		n, _, _ = tConn.ReadFromUDP(buf)

		cToSData := make([]byte, cfg.Client.PackSize)
		cToSData[0] = util.C_TO_S
		copy(cToSData[1:], sessionId)
		copy(cToSData[37:], buf[:n])
		length, _ = conn.Write(cToSData[:37+n])
		log.Printf("[%s] T -> S len=%d", sessionId, length-37)
	}
}

func main() {

	cfg = util.LoadConfig[Config]("./config/app.yml")

	go TCPClient()
	go UDPClient()

	wg.Add(1)
	wg.Wait()

}
