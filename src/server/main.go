package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync/atomic"

	"github.com/quic-go/quic-go"
)

var ConfigurationID byte = 0xA1
var globalQuestionCount int64 = 0

type ConnectionState struct {
	ClientName string
	PingCount  int
}

func main() {
	id := flag.Int("id", 0, "Server ID for cid")
	port := flag.Int("port", 443, "port")
	flag.Parse()
	if *id == 0 {
		fmt.Println("Use flag -id for launch server with custom ServerID. Now id is 0")
	}
	runServer(byte(*id), *port)
}

func runServer(serverID byte, port int) {
	tlsConf := generateTLSConfig()

	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		log.Fatalf("Failed to listen on UDP: %v", err)
	}

	transport := &quic.Transport{
		Conn:                  udpConn,
		ConnectionIDLength:    12,
		ConnectionIDGenerator: CreateCIDgen(ConfigurationID, serverID),
	}

	defer transport.Close()

	listener, err := transport.Listen(tlsConf, nil)
	if err != nil {
		log.Fatalf("Failed to start QUIC listener: %v", err)
	}
	defer listener.Close()

	log.Printf("QUIC Server %d is listening", serverID)

	for {
		ctx := context.Background()
		conn, err := listener.Accept(ctx)
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		state := &ConnectionState{}
		go handleClientConnection(conn, serverID, state)
	}
}

func handleClientConnection(conn *quic.Conn, serverID byte, state *ConnectionState) {
	log.Printf("[Server %d] New QUIC connection from %s", serverID, conn.RemoteAddr())

	defer conn.CloseWithError(0, "connection closed")

	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			log.Printf("[Server %d] Connection closed: %v", serverID, err)
			return
		}

		go func(s *quic.Stream) {
			defer s.Close()

			buf := make([]byte, 1024)
			n, err := s.Read(buf)
			if err != nil && err != io.EOF {
				log.Printf("[Server %d] Error reading: %v", serverID, err)
				return
			}

			msg := strings.TrimSpace(string(buf[:n]))
			var reply string

			if msg == "ping" {
				state.PingCount++
				reply = "pong"
			} else if msg == "gg" {
				reply = fmt.Sprintf("Well played. You did %d ping pong.", state.PingCount)
				state.PingCount = 0
			} else {
				state.PingCount = 0

				switch {
				case msg == "Hello! Who are you?":
					reply = fmt.Sprintf("Hello! I am Server №%d. Who are you?", serverID)

				case strings.HasPrefix(msg, "Good. I am "):
					name := strings.TrimSuffix(strings.TrimPrefix(msg, "Good. I am "), ".")
					state.ClientName = name
					reply = fmt.Sprintf("Nice to meet you, %s.", state.ClientName)

				case msg == "Where are you from?":
					serverAddr := conn.LocalAddr().String()
					clientAddr := conn.RemoteAddr().String()
					reply = fmt.Sprintf("I am from %s. Let me guess... you are from %s.", serverAddr, clientAddr)

				case msg == "How are you?":
					count := atomic.AddInt64(&globalQuestionCount, 1)
					reply = fmt.Sprintf("Good. Today %d people asked me this question.", count)

				case msg == "Let's play ping pong":
					reply = "Good idea."

				case msg == "bye":
					if state.ClientName != "" {
						reply = fmt.Sprintf("Bye %s!", state.ClientName)
					} else {
						reply = "Bye!"
					}
				case msg == "Good morning!":
					reply = "Good morning!)"

				default:
					reply = "Quack idk what do you mean"
				}
			}

			_, err = s.Write([]byte(reply))
			if err != nil {
				log.Printf("[Server %d] Error writing: %v", serverID, err)
			}
		}(stream)
	}
}
