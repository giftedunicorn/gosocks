package main

import (
	"log"
	"net"
	"os"
	"io"

	"github.com/giftedunicorn/gosocks/socks"
)

func main() {
	handleLocalConn()
}

func handleLocalConn() {
	const localAddr = ":8080"

	l, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Println("failed to listen", err.Error())
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("failed to accept: %s", err)
			continue
		}

		go func() {
			defer conn.Close()
			// io.Copy(conn, conn)
			// handleConnection(conn)
			tgt, err := socks.Handshake(conn)
			if err != nil {
				log.Println("failed to get target address: %s", err)
				continue
			}
			log.Println(tgt)
		}()
	}
}

func handleConnection(c net.Conn) {
	log.Println(c)
    buf := make([]byte, MaxAddrLen)

    for {
        n, err := c.Read(buf)
        if err != nil || n == 0 {
            c.Close()
            break
        }
        n, err = c.Write(buf[0:n])
        if err != nil {
            c.Close()
            break
        }
    }
    log.Printf("Connection from %v closed.\n", c.RemoteAddr())
}
