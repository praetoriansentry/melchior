// Package is the full implementation of a gemini protocol server
package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	Version = "0.0.1"
)

var (
	MelchiorTLSCert  = ""
	MelchiorTLSKey   = ""
	MelchiorHostname = ""
	MelchiorBindAddr = ""
	MelchiorRootDir  = ""
	MelchiorDeadline = 5
)

func initVars() error {
	MelchiorTLSCert = os.Getenv("MELCHIOR_TLS_CERT")
	MelchiorTLSKey = os.Getenv("MELCHIOR_TLS_KEY")
	MelchiorHostname = os.Getenv("MELCHIOR_HOSTNAME")
	MelchiorBindAddr = os.Getenv("MELCHIOR_BIND_ADDR")
	MelchiorRootDir = os.Getenv("MELCHIOR_ROOT_DIR")
	tmpMelchiorDeadline := os.Getenv("MELCHIOR_DEADLINE")

	if MelchiorTLSCert == "" || MelchiorTLSKey == "" {
		return fmt.Errorf("MELCHIOR_TLS_KEY and MELCHIOR_TLS_CERT must both be provided")
	}

	// Defaults
	if MelchiorHostname == "" {
		MelchiorHostname = "localhost"
	}
	if MelchiorBindAddr == "" {
		MelchiorBindAddr = "127.0.0.1:1965"
	}
	if MelchiorRootDir == "" {
		MelchiorRootDir = "."
	}
	if tmpMelchiorDeadline != "" {
		dl, err := strconv.Atoi(tmpMelchiorDeadline)
		if err == nil && dl > 0 {
			MelchiorDeadline = dl
		} else {
			log.Println("Unable to set deadline. Double check MELCHIOR_DEADLINE")
		}
	}

	log.Printf("The value of %s is %s", "MELCHIOR_TLS_CERT", MelchiorTLSCert)
	log.Printf("The value of %s is %s", "MELCHIOR_TLS_KEY", MelchiorTLSKey)
	log.Printf("The value of %s is %s", "MELCHIOR_HOSTNAME", MelchiorHostname)
	log.Printf("The value of %s is %s", "MELCHIOR_BIND_ADDR", MelchiorBindAddr)
	log.Printf("The value of %s is %s", "MELCHIOR_ROOT_DIR", MelchiorRootDir)
	log.Printf("The value of %s is %d", "MELCHIOR_DEADLINE", MelchiorDeadline)

	return nil
}

func main() {
	log.Printf("Starting Melchoir Version %s", Version)

	err := initVars()
	if err != nil {
		log.Fatalf("Failed to initialized %s", err)
	}

	cert, err := tls.LoadX509KeyPair(MelchiorTLSCert, MelchiorTLSKey)
	if err != nil {
		log.Fatalf("Unable to load TLS certficate: %s", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ServerName:   MelchiorHostname,
	}

	listener, err := tls.Listen("tcp", MelchiorBindAddr, tlsConfig)
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

		err = conn.SetDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			log.Printf("Unable to set connection deadline: %s", err)
		}

		go func() {
			err = handle(conn)
			if err != nil {
				log.Printf("There was an error handling the connection: %s", err)
			}

			err = conn.Close()
			if err != nil {
				log.Printf("There was an error closing the connection: %s", err)
			}
		}()
	}
}

func handle(conn net.Conn) error {
	log.Printf("Handling connection from %s connecting to us on %s", conn.RemoteAddr(), conn.LocalAddr())
	// The max length of one of these lines should be 1024 characters + \r\n
	requestData := make([]byte, 1026)
	readLen, err := conn.Read(requestData)
	if err != nil {
		return err
	}

	inputURL := requestData[0:readLen]

	urlString := strings.TrimSpace(string(inputURL))
	u, err := url.Parse(urlString)
	if err != nil {
		reply(conn, 59, "Bad URL")
		return err
	}

	log.Printf("Got a request to %s", urlString)

	uPath := u.Path

	if !strings.HasPrefix(uPath, "/") {
		uPath = "/" + uPath
	}

	// If we get a request to a folder, we're going to insert a call to index.gmi as the default file
	if strings.HasSuffix(uPath, "/") {
		uPath = uPath + "index.gmi"
	}

	// Even though we're not using HTTP we can leverage this
	// function to avoid directory traversal type attacks.
	var root http.Dir = http.Dir(MelchiorRootDir)

	cleanPath := filepath.Clean(uPath)
	log.Printf("Attempting to open file: %s", cleanPath)
	f, err := root.Open(cleanPath)
	if err != nil {
		reply(conn, 51, "File not found")
		return err
	}

	defer f.Close()

	body, err := ioutil.ReadAll(f)
	if err != nil {
		reply(conn, 59, "File read error")
		return err
	}

	meta := http.DetectContentType(body)

	if strings.HasSuffix(uPath, ".gmi") {
		meta = "text/gemini; lang=en; charset=utf-8"
	}

	_, err = fullResponse(conn, meta, body)

	if err != nil {
		return err
	}

	return nil
}

func reply(conn net.Conn, code int, message string) {
	msg := fmt.Sprintf("%d %s\r\n", code, message)
	_, err := conn.Write([]byte(msg))
	log.Printf("There was an error writing to the connection: %s", err)
}

func fullResponse(conn net.Conn, meta string, body []byte) (int, error) {
	reply(conn, 20, meta)
	return conn.Write([]byte(body))
}
