package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lucas-clemente/quic-go"
	"github.com/marten-seemann/quic-go-interop/http09"
)

var path string

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("runtime.Caller failed")
		os.Exit(1)
	}
	path = filepath.Dir(filename)
}

func main() {
	logFile, err := os.Create("/logs/log.txt")
	if err != nil {
		fmt.Printf("Could not create log file: %s\n", err.Error())
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	testcase := os.Getenv("TESTCASE")
	var quicConf *quic.Config
	switch testcase {
	case "handshake", "transfer":
		// don't use Retries
		quicConf = &quic.Config{
			AcceptToken: func(_ net.Addr, _ *quic.Token) bool { return true },
		}
	case "retry":
	default:
		fmt.Printf("unsupported test case: %s\n", testcase)
		os.Exit(127)
	}

	if err := runServer(quicConf); err != nil {
		fmt.Printf("Error running server: %s\n", err.Error())
		os.Exit(1)
	}
}

func runServer(quicConf *quic.Config) error {
	server := http09.Server{
		Server:     &http.Server{Addr: "0.0.0.0:443"},
		QuicConfig: quicConf,
	}
	http.DefaultServeMux.Handle("/", http.FileServer(http.Dir("/www")))
	return server.ListenAndServeQUIC(path+"/cert.pem", path+"/key.pem")
}
