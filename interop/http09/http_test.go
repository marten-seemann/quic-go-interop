package http09

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP 0.9 integration tests", func() {
	var (
		server *Server
		saddr  net.Addr
		done   chan struct{}
	)

	http.HandleFunc("/helloworld", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello World!"))
	})

	BeforeEach(func() {
		tlsConf, err := getTLSConfig()
		Expect(err).ToNot(HaveOccurred())
		server = &Server{
			Server: &http.Server{
				TLSConfig: tlsConf,
			},
		}
		addr, err := net.ResolveUDPAddr("udp", "localhost:0")
		Expect(err).ToNot(HaveOccurred())
		conn, err := net.ListenUDP("udp", addr)
		Expect(err).ToNot(HaveOccurred())
		saddr = conn.LocalAddr()
		done = make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			_ = server.Serve(conn)
		}()
	})

	AfterEach(func() {
		Expect(server.Close()).To(Succeed())
		Eventually(done).Should(BeClosed())
	})

	It("performs request", func() {
		rt := &RoundTripper{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("https://%s/helloworld", saddr),
			nil,
		)
		rsp, err := rt.RoundTrip(req)
		Expect(err).ToNot(HaveOccurred())
		data, err := ioutil.ReadAll(rsp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("Hello World!")))
	})

	It("allows setting of headers", func() {
		http.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("foo", "bar")
			w.WriteHeader(1337)
			_, _ = w.Write([]byte("done"))
		})

		rt := &RoundTripper{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("https://%s/headers", saddr),
			nil,
		)
		rsp, err := rt.RoundTrip(req)
		Expect(err).ToNot(HaveOccurred())
		data, err := ioutil.ReadAll(rsp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("done")))
	})
})
