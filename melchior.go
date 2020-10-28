package main

import (
	"log"
	"crypto/tls"
	"os"
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
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ServerName:   os.Getenv("MELCHIOR_HOSTNAME"),
	}
	log.Println(tlsConfig)
}
