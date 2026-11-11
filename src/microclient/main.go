package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/quic-go/quic-go"
)

func main() {
	serverIP := flag.String("addr", "localhost", "Server IP address")
	port := flag.String("port", "443", "Server port")
	flag.Parse()
	serverAddr := *serverIP + ":" + *port

	log.SetFlags(0)
	log.Printf("Connecting to %s...", serverAddr)

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-ping-pong"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := quic.DialAddr(ctx, serverAddr, tlsConf, nil)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.CloseWithError(0, "client finished")

	log.Printf("Connection established. Sending 'Good morning!'")

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		log.Fatalf("Failed to open stream: %v", err)
	}
	defer stream.Close()

	_, err = stream.Write([]byte("Good morning!"))
	if err != nil {
		log.Fatalf("Failed to write: %v", err)
	}

	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil && err != io.EOF {
		log.Fatalf("Failed to read: %v", err)
	}

	fmt.Printf("Server response: %s\n", string(buf[:n]))
}
