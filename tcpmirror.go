package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

const (
	BUFFSIZE = 4096
)

var (
	mirrorHost string
	mirrorPort = 80
)

func main() {
	println("tcpmirror starting ...")
	flag.Parse()
	if len(flag.Args()) != 2 {
		usage()
		os.Exit(1)
	}

	mirrorHost = flag.Args()[0]
	localPort, err := strconv.Atoi(flag.Args()[1])
	if err != nil {
		log.Fatalf("invalid port: %s", flag.Args()[1])
	}

	log.Printf("Mirroring tcp service from %s to 0.0.0.0:%d ...", mirrorHost, localPort)
	listenAddr := fmt.Sprintf(":%d", localPort)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		// handle error
		log.Fatalf("Listen failed: %s", err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			log.Printf("accept failed: %s", err)
			continue
		}
		log.Printf("connection accepted: %s", conn.RemoteAddr())
		go handleConnection(conn)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: tcpmirror <host> <port>\n")
}

func handleConnection(conn net.Conn) {
	// connecting to the remote host for the connection
	defer log.Printf("connection closed: %s", conn.RemoteAddr())
	defer conn.Close()

	mirrorConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", mirrorHost, mirrorPort))
	if err != nil {
		log.Printf("connect to mirror service failed: %s", err)
		return
	}

	defer mirrorConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go writer(conn, mirrorConn, &wg)
	reader(conn, mirrorConn, &wg)

	wg.Wait()
}

func reader(conn, mirrorConn net.Conn, wg *sync.WaitGroup) {
	readBuf := make([]byte, BUFFSIZE)
	for {
		n, err := conn.Read(readBuf)
		if err != nil {
			mirrorConn.Close()
			break
		}

		if n == 0 {
			mirrorConn.Close()
			break
		}

		mirrorConn.Write(readBuf[:n])
	}
	wg.Done()
}

func writer(conn, mirrorConn net.Conn, wg *sync.WaitGroup) {
	readBuf := make([]byte, BUFFSIZE)
	for {
		n, err := mirrorConn.Read(readBuf)
		if err != nil {
			conn.Close()
			break
		}

		if n == 0 {
			conn.Close()
			break
		}

		conn.Write(readBuf[:n])
	}
	wg.Done()
}
