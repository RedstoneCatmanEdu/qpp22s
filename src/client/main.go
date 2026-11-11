package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
)

type RebindingConn struct {
	mu   sync.RWMutex
	conn net.PacketConn
}

func NewRebindingConn() (*RebindingConn, error) {
	c, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}
	return &RebindingConn{conn: c}, nil
}

func (r *RebindingConn) RebindToIP(newIP string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	oldConn := r.conn
	newConn, err := net.ListenPacket("udp", newIP+":0")
	if err != nil {
		return err
	}
	r.conn = newConn
	oldConn.Close()
	return nil
}

func (r *RebindingConn) ReadFrom(p []byte) (int, net.Addr, error) {
	r.mu.RLock()
	c := r.conn
	r.mu.RUnlock()

	for {
		n, addr, err := c.ReadFrom(p)
		if err != nil {
			r.mu.RLock()
			newC := r.conn
			r.mu.RUnlock()
			if newC != c {
				return newC.ReadFrom(p)
			}
		}
		return n, addr, err
	}
}

func (r *RebindingConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.conn.WriteTo(p, addr)
}

func (r *RebindingConn) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conn.Close()
}

func (r *RebindingConn) LocalAddr() net.Addr {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.conn.LocalAddr()
}

func (r *RebindingConn) SetDeadline(t time.Time) error      { return nil }
func (r *RebindingConn) SetReadDeadline(t time.Time) error  { return nil }
func (r *RebindingConn) SetWriteDeadline(t time.Time) error { return nil }

func (r *RebindingConn) Rebind() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	oldConn := r.conn
	newConn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return err
	}
	r.conn = newConn
	oldConn.Close()
	return nil
}
func main() {
	serverIP := flag.String("addr", "localhost", "Server IP address")
	port := flag.String("port", "443", "Server port")
	Count := flag.Int("c", 1, "Number of simultaneous clients")
	changePort := flag.Bool("ChangePort", false, "Enable NAT Rebinding (Changesaurus mode)")
	changeIP := flag.String("ChangeIP", "", "New IP for IP Migration. empty = off")
	flag.Parse()

	serverAddrStr := fmt.Sprintf("%s:%s", *serverIP, *port)

	log.SetFlags(0)

	if *changeIP != "" {
		runChangeIPClient(serverAddrStr, *changeIP)
	} else if *changePort {

		runChangePortClient(serverAddrStr, *Count)
	} else {
		var wg sync.WaitGroup
		log.Printf("Starting %d clients simultaneously...\n\n", *Count)

		for i := 1; i <= *Count; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				clientName := generateName(index)
				runNormalClient(serverAddrStr, clientName)
			}(i)
		}

		wg.Wait()
		log.Printf("\nAll %d clients finished.\n", *Count)
	}
}

func generateName(index int) string {
	if index <= 26 {
		letter := rune('A' + index - 1)
		return fmt.Sprintf("%catur", letter)
	}
	return fmt.Sprintf("monster-ant-%d", index)
}

func runNormalClient(serverAddrStr string, clientName string) {
	conn, connWrap, cancel := connect(serverAddrStr)
	if conn == nil {
		return
	}
	defer cancel()
	defer conn.CloseWithError(0, "client finished")
	_ = connWrap

	serverName := "Server"

	log.Printf("[%s to %s]: Hello! Who are you?\n", clientName, serverName)
	reply := sendAndReceive(conn, "Hello! Who are you?")
	serverName = extractServerName(reply)
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	msg := fmt.Sprintf("Good. I am %s.", clientName)
	log.Printf("[%s to %s]: %s\n", clientName, serverName, msg)
	reply = sendAndReceive(conn, msg)
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	log.Printf("[%s to %s]: Where are you from?\n", clientName, serverName)
	reply = sendAndReceive(conn, "Where are you from?")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	log.Printf("[%s to %s]: Let's play ping pong\n", clientName, serverName)
	reply = sendAndReceive(conn, "Let's play ping pong")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	totalPings := rand.Intn(5) + 4
	for i := 1; i <= totalPings; i++ {
		log.Printf("[%s to %s]: ping\n", clientName, serverName)
		reply = sendAndReceive(conn, "ping")
		log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
	}

	log.Printf("[%s to %s]: gg\n", clientName, serverName)
	reply = sendAndReceive(conn, "gg")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	log.Printf("[%s to %s]: bye\n", clientName, serverName)
	reply = sendAndReceive(conn, "bye")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
}

func runChangePortClient(serverAddrStr string, count int) {
	clientName := "Changesaurus"
	conn, connWrap, cancel := connect(serverAddrStr)
	if conn == nil {
		return
	}
	defer cancel()
	defer conn.CloseWithError(0, "client finished")

	serverName := "Server"

	log.Printf("[%s to %s]: Hello! Who are you?\n", clientName, serverName)
	reply := sendAndReceive(conn, "Hello! Who are you?")
	serverName = extractServerName(reply)
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	msg := fmt.Sprintf("Good. I am %s.", clientName)
	log.Printf("[%s to %s]: %s\n", clientName, serverName, msg)
	reply = sendAndReceive(conn, msg)
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	log.Printf("[%s to %s]: Where are you from?\n", clientName, serverName)
	reply = sendAndReceive(conn, "Where are you from?")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
	log.Printf("[%s to %s]: Let's play ping pong\n", clientName, serverName)
	reply = sendAndReceive(conn, "Let's play ping pong")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
	log.Printf("[%s to %s]: ping\n", clientName, serverName)
	reply = sendAndReceive(conn, "ping")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	for i := 0; i < count; i++ {
		log.Printf("[!] NAT REBINDING: Connection dropped. Reconnecting on new port...\n")
		if err := connWrap.Rebind(); err != nil {
			log.Fatalf("Failed to rebind: %v", err)
		}
		time.Sleep(500 * time.Millisecond) // Let OVS update rules

		log.Printf("[%s to %s]: ping\n", clientName, serverName)
		reply = sendAndReceive(conn, "ping")
	}
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
	log.Printf("[%s to %s]: gg\n", clientName, serverName)
	reply = sendAndReceive(conn, "gg")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
	log.Printf("[%s to %s]: Hello! Who are you?\n", clientName, serverName)
	reply = sendAndReceive(conn, "Hello! Who are you?")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
	log.Printf("[%s to %s]: Where are you from?\n", clientName, serverName)
	reply = sendAndReceive(conn, "Where are you from?")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
	log.Printf("[%s to %s]: bye\n", clientName, serverName)
	reply = sendAndReceive(conn, "bye")
	log.Printf("[%s to %s]: %s\n\n", serverName, clientName, reply)
}

func runChangeIPClient(serverAddrStr string, newIP string) {
	clientName := "Migrasaurus"
	conn, connWrap, cancel := connect(serverAddrStr)
	if conn == nil {
		return
	}
	defer cancel()
	defer conn.CloseWithError(0, "client finished")

	serverName := "Server"

	log.Printf("[%s to %s]: Hello! Who are you?\n", clientName, serverName)
	reply := sendAndReceive(conn, "Hello! Who are you?")
	serverName = extractServerName(reply)
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	msg := fmt.Sprintf("Good. I am %s.", clientName)
	log.Printf("[%s to %s]: %s\n", clientName, serverName, msg)
	reply = sendAndReceive(conn, msg)
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	log.Printf("[%s to %s]: Where are you from?\n", clientName, serverName)
	reply = sendAndReceive(conn, "Where are you from?")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	log.Printf("[%s to %s]: Let's play ping pong\n", clientName, serverName)
	reply = sendAndReceive(conn, "Let's play ping pong")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	log.Printf("[%s to %s]: ping\n", clientName, serverName)
	reply = sendAndReceive(conn, "ping")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	log.Printf("[!] === MIGRATING IP + PORT -> %s ===\n", newIP)
	if err := connWrap.RebindToIP(newIP); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	time.Sleep(1 * time.Second)

	log.Printf("[%s to %s]: ping\n", clientName, serverName)
	reply = sendAndReceive(conn, "ping")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
	log.Printf("[%s to %s]: gg\n", clientName, serverName)
	reply = sendAndReceive(conn, "gg")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)
	log.Printf("[%s to %s]: Where are you from?\n", clientName, serverName)
	reply = sendAndReceive(conn, "Where are you from?")
	log.Printf("[%s to %s]: %s\n", serverName, clientName, reply)

	log.Printf("[%s to %s]: bye\n", clientName, serverName)
	reply = sendAndReceive(conn, "bye")
	log.Printf("[%s to %s]: %s\n\n", serverName, clientName, reply)
}

func connect(serverAddrStr string) (*quic.Conn, *RebindingConn, context.CancelFunc) {
	serverAddr, err := net.ResolveUDPAddr("udp", serverAddrStr)
	if err != nil {
		log.Printf("Failed to resolve address: %v\n", err)
		return nil, nil, nil
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-ping-pong"},
	}

	connWrap, err := NewRebindingConn()
	if err != nil {
		log.Printf("Failed to create socket: %v\n", err)
		return nil, nil, nil
	}

	transport := &quic.Transport{Conn: connWrap}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	conn, err := transport.Dial(ctx, serverAddr, tlsConf, nil)
	if err != nil {
		log.Printf("Failed to dial: %v\n", err)
		cancel()
		return nil, nil, nil
	}

	return conn, connWrap, cancel
}

func extractServerName(reply string) string {
	idx := strings.Index(reply, "Server №")
	if idx != -1 {
		end := strings.Index(reply[idx:], ".")
		if end != -1 {
			return reply[idx : idx+end]
		}
	}
	return "Server"
}

func sendAndReceive(conn *quic.Conn, message string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Sprintf("[Error opening stream: %v]", err)
	}
	defer stream.Close()

	_, err = stream.Write([]byte(message))
	if err != nil {
		return fmt.Sprintf("[Error writing: %v]", err)
	}

	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Sprintf("[Error reading: %v]", err)
	}

	return strings.TrimSpace(string(buf[:n]))
}
