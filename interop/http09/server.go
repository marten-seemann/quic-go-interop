package http09

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"

	"github.com/lucas-clemente/quic-go"
)

const h09alpn = "h09"

type responseWriter struct {
	io.Writer
	headers http.Header
}

var _ http.ResponseWriter = &responseWriter{}

func (w *responseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *responseWriter) WriteHeader(int) {}

type Server struct {
	*http.Server

	QuicConfig *quic.Config

	mutex    sync.Mutex
	listener quic.Listener
}

func (s *Server) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.listener.Close()
}

func (s *Server) ListenAndServeQUIC(certFile, keyFile string) error {
	var err error
	certs := make([]tls.Certificate, 1)
	certs[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	s.TLSConfig = &tls.Config{Certificates: certs}

	udpAddr, err := net.ResolveUDPAddr("udp", s.Addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	return s.Serve(conn)
}

func (s *Server) Serve(conn net.PacketConn) error {
	tlsConf := s.TLSConfig.Clone()
	tlsConf.NextProtos = []string{h09alpn}
	ln, err := quic.Listen(conn, tlsConf, s.QuicConfig)
	if err != nil {
		return err
	}
	s.mutex.Lock()
	s.listener = ln
	s.mutex.Unlock()

	for {
		sess, err := ln.Accept(context.Background())
		if err != nil {
			return err
		}
		go s.handleConn(sess)
	}
}

func (s *Server) handleConn(sess quic.Session) {
	for {
		str, err := sess.AcceptStream(context.Background())
		if err != nil {
			log.Printf("Error accepting stream: %s\n", err.Error())
			return
		}
		go func() {
			if err := s.handleStream(str); err != nil {
				log.Printf("Handling stream failed: %s", err.Error())
			}
		}()
	}
}

func (s *Server) handleStream(str quic.Stream) error {
	reqBytes, err := ioutil.ReadAll(str)
	if err != nil {
		return err
	}
	request := string(reqBytes)
	request = strings.TrimRight(request, "\r\n")
	request = strings.TrimRight(request, " ")
	if request[:5] != "GET /" {
		str.CancelWrite(42)
		return nil
	}

	split := strings.Split(request[4:], " ")
	if len(split) > 2  || (len(split) == 2 && split[1] != "HTTP/0.9") {
		return nil
	}
	u, err := url.Parse(split[0])
	if err != nil {
		return err
	}
	u.Scheme = "https"

	req := &http.Request{
		Method:     http.MethodGet,
		Proto:      "HTTP/0.9",
		ProtoMajor: 0,
		ProtoMinor: 9,
		Body:       str,
		URL:        u,
	}

	handler := s.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}

	var panicked bool
	func() {
		defer func() {
			if p := recover(); p != nil {
				// Copied from net/http/server.go
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				log.Printf("http: panic serving: %v\n%s", p, buf)
				panicked = true
			}
		}()
		handler.ServeHTTP(&responseWriter{Writer: str}, req)
	}()

	if panicked {
		if _, err := str.Write([]byte("500")); err != nil {
			return err
		}
	}
	return str.Close()
}
