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
	"strings"
	"time"
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
		err = conn.SetDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			log.Printf("Unable to set connection deadline: %s", err)
		}

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

	urlString := strings.TrimSpace(string(inputURL))
	u, err := url.Parse(urlString)
	if err != nil {
		reply(conn, 59, "Bad URL")
		return err
	}

	uPath := u.Path

	if !strings.HasPrefix(uPath, "/") {
		uPath = "/" + uPath
	}

	if strings.HasSuffix(uPath, "/") {
		uPath = uPath + "index.gmi"
	}

	var root http.Dir = http.Dir(os.Getenv("MELCHIOR_ROOT_DIR"))

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

func reply(conn net.Conn, code int, message string) (int, error) {
	msg := fmt.Sprintf("%d %s\r\n", code, message)
	return conn.Write([]byte(msg))
}
func fullResponse(conn net.Conn, meta string, body []byte) (int, error) {
	_, err := reply(conn, 20, meta)
	if err != nil {
		return 0, err
	}
	return conn.Write([]byte(body))
}
