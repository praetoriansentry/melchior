package main

import (
	"log"
	"crypto/tls"
	// 	"time"
	"os"
	"net"
)
const (
	Version = "0.0.1"
)

func main() {
	log.Printf("Starting Melchoir Version %s", Version)

	cert, err := tls.LoadX509KeyPair(os.Getenv("MELCHIOR_TLS_CERT"), os.Getenv("MELCHIOR_TLS_KEY"))
	if err != nil {
		log.Fatalf("Unable to load TLS certficate: %s", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// ClientAuth:   tls.VerifyClientCertIfGiven,
		// ServerName:   os.Getenv("MELCHIOR_HOSTNAME"),
	}

	listener, err := tls.Listen("tcp", os.Getenv("MELCHIOR_BIND_ADDR"), tlsConfig)
	if err != nil {
		log.Fatalf("Unable to listen: %s", err)
	}
	log.Println("Started listening")
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("There was an error accepting the connection %s", err)
		}

		// todo make this configuratable
		// err = conn.SetDeadline(time.Now().Add(5 * time.Second))
		// if err != nil {
		// 	log.Printf("Unable to set connection deadline: %s", err)
		// }

		err = handle(conn)
		if err != nil {
			log.Printf("There was an error handling the connection: %s", err)
		}

		err = conn.Close()
		if err != nil {
			log.Printf("There was an error closing the connection: %s", err)
		}
	}
}

func handle(conn net.Conn) error {
	log.Printf("Handling connection from %s connecting to us on %s", conn.RemoteAddr(), conn.LocalAddr())
	requestData := make([]byte, 1026)
	readLen, err := conn.Read(requestData)
	if err != nil {
		return err
	}
	inputURL := requestData[0:readLen]
	log.Println(string(inputURL))
	conn.Write([]byte("20 text/plain\r\nhi there\r\n"))
	return nil
}
