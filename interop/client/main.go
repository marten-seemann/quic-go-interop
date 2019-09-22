package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/marten-seemann/quic-go-interop/http09"
	"golang.org/x/sync/errgroup"
)

func main() {
	flag.Parse()
	urls := flag.Args()

	testcase := os.Getenv("TESTCASE")
	switch testcase {
	case "handshake", "transfer", "retry":
	default:
		fmt.Printf("unsupported test case: %s\n", testcase)
		os.Exit(127)
	}

	roundTripper := &http09.RoundTripper{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	defer roundTripper.Close()

	var g errgroup.Group
	for _, u := range urls {
		url := u
		g.Go(func() error {
			return downloadFile(roundTripper, url)
		})
	}
	if err := g.Wait(); err != nil {
		panic(err)
	}
}

func downloadFile(cl http.RoundTripper, url string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	rsp, err := cl.RoundTrip(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	file, err := os.Create("/downloads" + req.URL.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, rsp.Body)
	return err
}
