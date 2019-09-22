package main

import (
	"fmt"
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
		panic("runtime.Caller failed")
	}
	path = filepath.Dir(filename)
}

func main() {
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
		panic(err)
	}
}

func runServer(quicConf *quic.Config) error {
	http.DefaultServeMux.Handle("/", http.FileServer(http.Dir("/www")))
	return http09.ListenAndServeQUIC("0.0.0.0:443", path+"/cert.pem", path+"/key.pem", nil)
}
