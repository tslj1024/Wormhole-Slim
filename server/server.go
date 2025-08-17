package main

import (
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"util"
)

// Define configuration
type Config struct {
	Server struct {
		Port    string `yaml:"port"`
		Clients []struct {
			ClientID string `yaml:"clientId"`
			Port     string `yaml:"port"`
			THost    string `yaml:"tHost"`
			TPort    string `yaml:"tPort"`
			Protocol string `yaml:"protocol"`
		} `yaml:"clients"`
		BufSize  int `yaml:"bufSize"`
		PackSize int `yaml:"packSize"`
	} `yaml:"server"`
}

// wg Used to wait for all coroutines to finish.
var wg sync.WaitGroup

var rwLock sync.RWMutex

// Used to store the mapping of ClientID and the corresponding signaling channel for that ClientID.
var clientConnMap map[string]*net.TCPConn

// Cache each session of the user
// Each time the user accesses the server, a SessionID is assigned and a TCP connection is established.
// This mapping stores the relationship between the SessionID and the TCP connection.
var sessionConnMap map[string]*net.TCPConn

// Configuration
var cfg *Config

// sessionId -> listener
var sessionListenerMap map[string]*net.UDPConn

// sessionId -> UserAddr
var sessionAddrMap map[string]*net.UDPAddr

// clientId -> ClientAddr
var clientAddrMap map[string]*net.UDPAddr

// Listen to the client's UDP packets
var clientUDPConn *net.UDPConn

// initialize
func initialize() {

	cfg = util.LoadConfig[Config]("./config/app.yml")

	clientConnMap = make(map[string]*net.TCPConn)
	sessionConnMap = make(map[string]*net.TCPConn)

	sessionListenerMap = make(map[string]*net.UDPConn)
	sessionAddrMap = make(map[string]*net.UDPAddr)
	clientAddrMap = make(map[string]*net.UDPAddr)

	var err error
	clientUDPConn, err = util.CreateUDPListen("udp", "", cfg.Server.Port)
	if err != nil {
		log.Fatalf("CreateUDPListen Error: %s\n", err.Error())
	}
	log.Printf("Server Started at %s", clientUDPConn.LocalAddr())
}

// TCP Methods

// clientListen Listen to client registration
func clientListen() {
	tcpListener, err := util.CreateTCPListen("tcp", "", cfg.Server.Port)
	if err != nil {
		log.Printf("Client listen Error: %s\n", err.Error())
		panic(err)
		return
	}
	log.Printf("Client Listen SUCCESS: %s\n", tcpListener.Addr().String())
	log.Printf("Waiting for client connection...\n")
	for {
		serverConn, err1 := tcpListener.AcceptTCP()
		if err1 != nil {
			log.Printf("Client connection Error：%s\n", err1.Error())
			continue
		}
		go handleClientTCPConn(serverConn)
	}
}

func handleClientTCPConn(conn *net.TCPConn) {
	for {
		data, err := util.GetDataFromConnection(cfg.Server.BufSize, conn)
		if err != nil {
			log.Printf("Client data read Error：%s\n", conn.RemoteAddr().String())
			return
		}
		flag := data[0]
		if flag == util.CONNECT {
			f1 := false
			for _, client := range cfg.Server.Clients {
				if client.ClientID == string(data[1:]) {
					rwLock.Lock()
					clientConnMap[string(data[1:])] = conn
					rwLock.Unlock()
					f1 = true
				}
			}
			if !f1 {
				// ClientID is not registered or in valid
				_ = conn.Close()
				return
			}
			log.Printf("Client connection SUCCESS：%s\n", conn.RemoteAddr().String())
		} else if flag == util.HEARTBEAT {
			// PONG
			_, err1 := conn.Write([]byte{util.HEARTBEAT})
			if err1 != nil {
				return
			}
		} else if flag == util.C_TO_S {
			rwLock.RLock()
			uconn := sessionConnMap[string(data[1:])]
			delete(sessionConnMap, string(data[1:]))
			rwLock.RUnlock()
			go func() {
				n, _ := io.Copy(conn, uconn)
				log.Printf("[%s] U -> C len= %d B", string(data[1:]), n)
			}()
			go func() {
				n, _ := io.Copy(uconn, conn)
				log.Printf("[%s] C -> U len= %d B", string(data[1:]), n)
			}()
			return
		}
	}
}

// portListen Listen to user connections from the ports specified in the configuration file.
func portTCPListen(network string, port string, clientID string, thost string, tport string) {
	tcpListener, err := util.CreateTCPListen(network, "", port)
	if err != nil {
		log.Printf("User listen Error:%s\n", err)
		panic(err)
		return
	}
	log.Printf("User listen SUCCESS：%s\n", tcpListener.Addr().String())
	for {
		userConn, err1 := tcpListener.AcceptTCP()
		if err1 != nil {
			log.Printf("User connect Error：%s\n", err1.Error())
			return
		}
		log.Printf("User connect SUCCESS：%s\n", userConn.RemoteAddr().String())
		sessionID := util.GenerateUUID()
		rwLock.Lock()
		sessionConnMap[sessionID] = userConn
		rwLock.Unlock()

		// Notify the client to establish a data channel.
		// sessionID:IP:Port
		data := make([]byte, cfg.Server.BufSize)
		data[0] = util.S_TO_C
		thostLen := byte(len(thost))
		tportLen := byte(len(tport))
		copy(data[1:], sessionID)
		data[37] = thostLen
		copy(data[38:], thost)
		data[38+thostLen] = tportLen
		copy(data[39+thostLen:], tport)
		rwLock.RLock()
		_, err2 := clientConnMap[clientID].Write(data[:39+thostLen+tportLen])
		rwLock.RUnlock()
		if err2 != nil {
			log.Printf("Send msg Error：%s\n", err2.Error())
		}
	}
}

// UDP Methods

func handleConn(conn *net.UDPConn) {
	buf := make([]byte, cfg.Server.PackSize)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if n == 0 || err != nil {
			continue
		}
		if buf[0] == util.CONNECT {
			clientId := string(buf[1:n])
			for _, client := range cfg.Server.Clients {
				if client.ClientID == clientId {
					rwLock.Lock()
					clientAddrMap[client.ClientID] = addr
					log.Printf("UDPClient connect success: [%s] %s", clientId, addr.String())
					rwLock.Unlock()
				}
			}
			// 非法ClientID
		} else if buf[0] == util.C_TO_S {
			sessionId := string(buf[1:37])
			rwLock.RLock()
			length := 0
			length, _ = sessionListenerMap[sessionId].WriteToUDP(buf[37:n], sessionAddrMap[sessionId])
			log.Printf("[%s] C -> U len=%d", sessionId, length)
			delete(sessionListenerMap, sessionId)
			delete(sessionAddrMap, sessionId)
			rwLock.RUnlock()
		}
	}
}

func portUDPListen(network string, port string, clientID string, thost string, tport string) {
	listener, err := util.CreateUDPListen(network, "", port)
	if err != nil {
		log.Fatalf("User listen Error: %s\n", err)
		return
	}
	log.Printf("User listen SUCCESS：%s\n", listener.LocalAddr().String())
	buf := make([]byte, cfg.Server.PackSize)
	for {
		n, userAddr, _ := listener.ReadFromUDP(buf)
		sessionID := util.GenerateUUID()

		rwLock.Lock()
		sessionListenerMap[sessionID] = listener
		sessionAddrMap[sessionID] = userAddr
		rwLock.Unlock()

		data := make([]byte, cfg.Server.PackSize)
		data[0] = util.S_TO_C
		thostLen := byte(len(thost))
		tportLen := byte(len(tport))
		copy(data[1:], sessionID)
		data[37] = thostLen
		copy(data[38:], thost)
		data[38+thostLen] = tportLen
		copy(data[39+thostLen:], tport)
		copy(data[39+thostLen+tportLen:], buf[:n])

		rwLock.RLock()
		_, _ = clientUDPConn.WriteToUDP(data[:int(39+thostLen+tportLen)+n], clientAddrMap[clientID])
		log.Printf("[%s] U -> C len=%d", sessionID, n)
		rwLock.RUnlock()
	}
}

func main() {

	initialize()

	go clientListen()
	go handleConn(clientUDPConn)

	for _, client := range cfg.Server.Clients {
		protocol := strings.Split(client.Protocol, "/")
		for _, v := range protocol {
			if strings.HasPrefix(v, "tcp") {
				go portTCPListen(v, client.Port, client.ClientID, client.THost, client.TPort)
			} else if strings.HasPrefix(v, "udp") {
				go portUDPListen(v, client.Port, client.ClientID, client.THost, client.TPort)
			}
		}
	}

	wg.Add(1)
	wg.Wait()
}
