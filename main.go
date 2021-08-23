package main

import (
	"log"
	"fmt"
	"net"
	"os"
	"io"

	// "github.com/giftedunicorn/gosocks/socks"
	"github.com/armon/go-socks5"
)

func main() {
	go createSocksServer()
	handleLocalConn()
}

func createSocksServer() {
	// Create a SOCKS5 server
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
	  panic(err)
	}

	// Create SOCKS5 proxy on localhost port 8000
	if err := server.ListenAndServe("tcp", "127.0.0.1:8081"); err != nil {
	  panic(err)
	}
}

func handleLocalConn() {
	const localAddr = ":8080"
	const server = ":8081"

	l, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Println("failed to listen", err.Error())
		os.Exit(1)
	}
	fmt.Printf("http://%v -> %s\n", l.Addr(), server)

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("failed to accept: %s", err)
			continue
		}

		fmt.Printf("Connection from %s\n", c.RemoteAddr())
		go func() {
			defer c.Close()
			// tgt, err := socks.Handshake(c)
			// if err != nil {
			// 	log.Println("failed to get target address: %s", err)
			// }
			// log.Println("target address", string(tgt))

			rc, err := net.Dial("tcp", server)
			if err != nil {
				log.Println("failed to connect to server %v: %v", server, err)
				os.Exit(1)
			}

			defer rc.Close()
			go io.Copy(rc, c)
			io.Copy(c, rc)
			fmt.Printf("Disconnect from %s\n", c.RemoteAddr())
		}()
	}
}

// https://stackoverflow.com/questions/32135763/is-it-possible-to-transport-a-tcp-connection-over-websockets-or-another-protocol
func run(inPort int, dest string) {
    l, err := net.Listen("tcp", fmt.Sprintf(":%d", inPort))
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    fmt.Printf("http://%v -> %s\n", l.Addr(), dest)

    for {
        conn, err := l.Accept()
        if err != nil {
            log.Fatal(err)
        }

        fmt.Printf("Connection from %s\n", conn.RemoteAddr())
        go proxy(conn, dest)
    }
}

func proxy(in net.Conn, dest string) {
    out, err := net.Dial("tcp", dest)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    go io.Copy(out, in)
    io.Copy(in, out)
    fmt.Printf("Disconnect from %s\n", in.RemoteAddr())
}
