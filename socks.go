// Package socks implements essential parts of SOCKS protocol.
package socks

import (
    "io"
    "net"
    "strconv"
)

// UDPEnabled is the toggle for UDP support
var UDPEnabled = false

// SOCKS request commands as defined in RFC 1928 section 4.
const (
    CmdConnect      = 1
    CmdBind         = 2
    CmdUDPAssociate = 3
)

// SOCKS address types as defined in RFC 1928 section 5.
const (
    AtypIPv4       = 1
    AtypDomainName = 3
    AtypIPv6       = 4
)

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

// MaxAddrLen is the maximum size of SOCKS address in bytes.
const MaxAddrLen = 1 + 1 + 255 + 2

// Addr represents a SOCKS address as defined in RFC 1928 section 5.
type Addr []byte

func readAddr(r io.Reader, b []byte) (Addr, error) {
    if len(b) < MaxAddrLen {
        return nil, io.ErrShortBuffer
    }
    _, err := io.ReadFull(r, b[:1]) // read 1st byte for address type
    if err != nil {
        return nil, err
    }

    switch b[0] {
    case AtypDomainName:
        _, err = io.ReadFull(r, b[1:2]) // read 2nd byte for domain length
        if err != nil {
            return nil, err
        }
        _, err = io.ReadFull(r, b[2:2+int(b[1])+2])
        return b[:1+1+int(b[1])+2], err
    case AtypIPv4:
        _, err = io.ReadFull(r, b[1:1+net.IPv4len+2])
        return b[:1+net.IPv4len+2], err
    case AtypIPv6:
        _, err = io.ReadFull(r, b[1:1+net.IPv6len+2])
        return b[:1+net.IPv6len+2], err
    }

    return nil, ErrAddressNotSupported
}

// Handshake fast-tracks SOCKS initialization to get target address to connect.
func Handshake(rw io.ReadWriter) (Addr, error) {
    // Read RFC 1928 for request and reply structure and sizes.
    buf := make([]byte, MaxAddrLen)
    // read VER, NMETHODS, METHODS
    if _, err := io.ReadFull(rw, buf[:2]); err != nil {
        return nil, err
    }
    nmethods := buf[1]
    if _, err := io.ReadFull(rw, buf[:nmethods]); err != nil {
        return nil, err
    }
    // write VER METHOD
    if _, err := rw.Write([]byte{5, 0}); err != nil {
        return nil, err
    }
    // read VER CMD RSV ATYP DST.ADDR DST.PORT
    if _, err := io.ReadFull(rw, buf[:3]); err != nil {
        return nil, err
    }
    cmd := buf[1]
    addr, err := readAddr(rw, buf)
    if err != nil {
        return nil, err
    }
    switch cmd {
    case CmdConnect:
        _, err = rw.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}) // SOCKS v5, reply succeeded
    case CmdUDPAssociate:
        if !UDPEnabled {
            return nil, ErrCommandNotSupported
        }
        listenAddr := ParseAddr(rw.(net.Conn).LocalAddr().String())
        _, err = rw.Write(append([]byte{5, 0, 0}, listenAddr...)) // SOCKS v5, reply succeeded
        if err != nil {
            return nil, ErrCommandNotSupported
        }
        err = InfoUDPAssociate
    default:
        return nil, ErrCommandNotSupported
    }

    return addr, err // skip VER, CMD, RSV fields
}
