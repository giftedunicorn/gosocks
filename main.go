package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	// "github.com/giftedunicorn/gosocks/socks"
	"github.com/armon/go-socks5"
)

const localAddr = ":8080"
const server = ":8081"

type Addr []byte

// Error represents a SOCKS error
type Error byte

func (err Error) Error() string {
	return "SOCKS error: " + strconv.Itoa(int(err))
}

// SOCKS errors as defined in RFC 1928 section 6.
const (
	ErrGeneralFailure       = Error(1)
	ErrConnectionNotAllowed = Error(2)
	ErrNetworkUnreachable   = Error(3)
	ErrHostUnreachable      = Error(4)
	ErrConnectionRefused    = Error(5)
	ErrTTLExpired           = Error(6)
	ErrCommandNotSupported  = Error(7)
	ErrAddressNotSupported  = Error(8)
	InfoUDPAssociate        = Error(9)
)

func main() {
	// go createSocksServer()
	startLocal()
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

func startLocal() {
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
		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	defer c.Close()

	tgt, err := handshake(c)
	if err != nil {
		log.Println("Socks handshake failed:", err)
		return
	}
	log.Println("target address", string(tgt))

	rc, err := net.Dial("tcp", server)
	if err != nil {
		log.Println("failed to connect to server %v: %v", server, err)
		return
	}
	defer rc.Close()

	// todo, send encrypted packets

	// reply
	go io.Copy(rc, c)
	io.Copy(c, rc)
	fmt.Printf("Disconnect from %s\n", c.RemoteAddr())
}

func handshake(c net.Conn) (Addr, error) {
	buf := make([]byte, 258)
	// read VER, NMETHODS, METHODS
	if _, err := io.ReadFull(c, buf[:2]); err != nil {
		return nil, err
	}
	nmethods := buf[1]
	if _, err := io.ReadFull(c, buf[:nmethods]); err != nil {
		return nil, err
	}
	// write VER METHOD
	if _, err := c.Write([]byte{5, 0}); err != nil {
		return nil, err
	}
	// read VER CMD RSV ATYP DST.ADDR DST.PORT
	if _, err := io.ReadFull(c, buf[:3]); err != nil {
		return nil, err
	}
	cmd := buf[1]
	addr, err := readAddr(c, buf)
	if err != nil {
		return nil, err
	}

	switch cmd {
	case 1:
		_, err = c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}) // SOCKS v5, reply succeeded
	case 3:

	default:
		return nil, Error(7)
	}

	log.Println("addr", addr)
	return addr, err // skip VER, CMD, RSV fields
}

func readAddr(r io.Reader, b []byte) (Addr, error) {
	if len(b) < 258 {
		return nil, io.ErrShortBuffer
	}
	_, err := io.ReadFull(r, b[:1]) // read 1st byte for address type
	if err != nil {
		return nil, err
	}

	switch b[0] {
	case 3:
		_, err = io.ReadFull(r, b[1:2]) // read 2nd byte for domain length
		if err != nil {
			return nil, err
		}
		_, err = io.ReadFull(r, b[2:2+int(b[1])+2])
		return b[:1+1+int(b[1])+2], err
	case 1:
		_, err = io.ReadFull(r, b[1:1+net.IPv4len+2])
		return b[:1+net.IPv4len+2], err
	case 4:
		_, err = io.ReadFull(r, b[1:1+net.IPv6len+2])
		return b[:1+net.IPv6len+2], err
	}

	return nil, Error(8)
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
