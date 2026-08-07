package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	ros "github.com/go-routeros/routeros/v3"
)

func main() {
	addr := "192.168.54.1:8729"

	// Test 1: Raw TCP probe
	fmt.Println("=== Test 1: Raw TCP probe ===")
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		fmt.Printf("TCP FAIL: %v\n", err)
		return
	}
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	fmt.Printf("Received %d bytes\n", n)
	if n > 0 {
		fmt.Printf("First byte: 0x%02x\n", buf[0])
		if buf[0] == 0x16 {
			fmt.Println("=> TLS ServerHello detected")
		}
	}
	conn.Close()

	// Test 2: TLS handshake with Go default
	fmt.Println("\n=== Test 2: TLS handshake (Go default) ===")
	conn2, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		fmt.Printf("TCP FAIL: %v\n", err)
		return
	}
	tlsConn := tls.Client(conn2, &tls.Config{InsecureSkipVerify: true})
	tlsConn.SetDeadline(time.Now().Add(5 * time.Second))
	err = tlsConn.Handshake()
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
	} else {
		state := tlsConn.ConnectionState()
		fmt.Printf("OK! Version=0x%x Cipher=0x%x (%s)\n", state.Version, state.CipherSuite, tls.CipherSuiteName(state.CipherSuite))
	}
	conn2.Close()

	// Test 3: TLS with MinVersion TLS1.0
	fmt.Println("\n=== Test 3: TLS handshake (TLS 1.0) ===")
	conn3, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		fmt.Printf("TCP FAIL: %v\n", err)
		return
	}
	tlsConn3 := tls.Client(conn3, &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS10,
		MaxVersion:         tls.VersionTLS10,
	})
	tlsConn3.SetDeadline(time.Now().Add(5 * time.Second))
	err = tlsConn3.Handshake()
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
	} else {
		state := tlsConn3.ConnectionState()
		fmt.Printf("OK! Version=0x%x Cipher=0x%x (%s)\n", state.Version, state.CipherSuite, tls.CipherSuiteName(state.CipherSuite))
	}
	conn3.Close()

	// Test 4: ros.DialTLSTimeout with default
	fmt.Println("\n=== Test 4: ros.DialTLSTimeout (default TLS) ===")
	_, err = ros.DialTLSTimeout(addr, "noc", "test", &tls.Config{InsecureSkipVerify: true}, 8*time.Second)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
	} else {
		fmt.Println("OK")
	}

	// Test 5: ros.DialTimeout (plain TCP)
	fmt.Println("\n=== Test 5: ros.DialTimeout (plain TCP) ===")
	_, err = ros.DialTimeout(addr, "noc", "test", 8*time.Second)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
	} else {
		fmt.Println("OK")
	}
}
