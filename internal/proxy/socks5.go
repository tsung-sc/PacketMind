package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/packetmind/packetmind/internal/storage"
)

func (p *Proxy) socks5Handshake(conn net.Conn) error {
	buf := make([]byte, 257)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}

	if n < 2 || buf[0] != socks5Version {
		return fmt.Errorf("invalid SOCKS version")
	}

	nmethods := int(buf[1])
	if n < 2+nmethods {
		return fmt.Errorf("invalid handshake length")
	}

	hasNoAuth := false
	for i := 0; i < nmethods; i++ {
		if buf[2+i] == socks5NoAuth {
			hasNoAuth = true
			break
		}
	}

	if !hasNoAuth {
		conn.Write([]byte{socks5Version, 0xFF})
		return fmt.Errorf("no acceptable auth method")
	}

	_, err = conn.Write([]byte{socks5Version, socks5NoAuth})
	return err
}

func (p *Proxy) socks5ReadRequest(conn net.Conn) (string, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	if buf[0] != socks5Version || buf[1] != socks5Connect {
		return "", fmt.Errorf("unsupported command")
	}

	var targetAddr string
	var port uint16

	switch buf[3] {
	case socks5AtypIPv4:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", err
		}
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, portBuf); err != nil {
			return "", err
		}
		port = binary.BigEndian.Uint16(portBuf)
		targetAddr = fmt.Sprintf("%d.%d.%d.%d:%d", ip[0], ip[1], ip[2], ip[3], port)

	case socks5AtypDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", err
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, portBuf); err != nil {
			return "", err
		}
		port = binary.BigEndian.Uint16(portBuf)
		targetAddr = fmt.Sprintf("%s:%d", string(domain), port)

	case socks5AtypIPv6:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", err
		}
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, portBuf); err != nil {
			return "", err
		}
		port = binary.BigEndian.Uint16(portBuf)
		targetAddr = fmt.Sprintf("[%s]:%d", net.IP(ip).String(), port)

	default:
		return "", fmt.Errorf("unsupported address type")
	}

	return targetAddr, nil
}

func (p *Proxy) socks5SendReply(conn net.Conn, rep byte) error {
	resp := []byte{
		socks5Version,
		rep,
		0,
		socks5AtypIPv4,
		0, 0, 0, 0,
		0, 0,
	}
	_, err := conn.Write(resp)
	return err
}

// recordSOCKS5Request creates and saves a request record for SOCKS5 connections
func (p *Proxy) recordSOCKS5Request(targetAddr, remoteAddr string, startTime time.Time, statusCode int, reqBytes, respBytes int64) {
	host, port := parseHostPort(targetAddr)

	record := &storage.Request{
		ID:          uuid.New().String(),
		CreatedAt:   startTime,
		Method:      "CONNECT",
		Scheme:      "socks5",
		URL:         "socks5://" + targetAddr,
		Host:        host,
		Port:        port,
		Path:        "",
		HTTPVersion: "SOCKS5",
		Headers:     make(storage.Headers),
		ContentType: "",
		Body:        nil,
		BodySize:    reqBytes,

		StatusCode:      statusCode,
		StatusReason:    "SOCKS5 Tunnel",
		RespHeaders:     make(storage.Headers),
		RespContentType: "",
		RespBody:        nil,
		RespBodySize:    respBytes,

		Duration:          time.Since(startTime).Milliseconds(),
		RemoteAddr:        remoteAddr,
		ClientAddr:        remoteAddr,
		ServerAddr:        targetAddr,
		RequestStartTime:  startTime,
		RequestEndTime:    startTime,
		ResponseStartTime: time.Now(),
		ResponseEndTime:   time.Now(),
		ResponseDuration:  time.Since(startTime).Milliseconds(),
	}

	p.saveRequestStart(record)
}

func (p *Proxy) tunnelSOCKS5(client net.Conn, clientReader io.Reader, target net.Conn, targetAddr string, startTime time.Time) {
	var reqBytes, respBytes int64
	done := make(chan struct{}, 2)

	// Client -> Target
	go func() {
		n, _ := io.Copy(target, clientReader)
		reqBytes = n
		done <- struct{}{}
	}()

	// Target -> Client
	go func() {
		n, _ := io.Copy(client, target)
		respBytes = n
		done <- struct{}{}
	}()

	// Wait for one direction to finish
	<-done
	client.Close()
	target.Close()

	// Wait for the other direction
	<-done

	// Record the SOCKS5 request with byte counts
	p.recordSOCKS5Request(targetAddr, client.RemoteAddr().String(), startTime, http.StatusOK, reqBytes, respBytes)
}
