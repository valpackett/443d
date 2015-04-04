package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/bradfitz/http2"
	"github.com/myfreeweb/443d/demux"
)

var sshHost = flag.String("ssh", "localhost:22", "hostname:port of the SSH server")
var httpNetwork = flag.String("httpnet", "tcp", "network of the HTTP/1.1 server")
var httpHost = flag.String("http", "localhost:8080", "address (hostname:port or path if httpnet is unix) of the HTTP/1.1 server")
var listenHost = flag.String("listen", "0.0.0.0:443", "hostname:port to listen on")
var crtPath = flag.String("crt", "server.crt", "path to the TLS certificate")
var keyPath = flag.String("key", "server.key", "path to the TLS private key")

func main() {
	flag.Parse()
	httph := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = "http"
			r.URL.Host = *httpHost
		},
		Transport: &http.Transport{
			Dial: func(network string, address string) (net.Conn, error) {
				if *httpNetwork == "unix" {
					address = address[:len(address)-3] // remove :80
				}
				return net.Dial(*httpNetwork, address)
			},
		},
	}
	sshh := demux.SshHandler(*sshHost)
	srv := &http.Server{
		Addr:    *listenHost,
		Handler: httph,
	}
	http2.ConfigureServer(srv, &http2.Server{})
	srv.TLSConfig.Certificates = make([]tls.Certificate, 1)
	srv.TLSConfig.Certificates[0], _ = tls.LoadX509KeyPair(*crtPath, *keyPath)
	tcpl, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Fatalf("%v :-(\n", err)
	}
	dl := &demux.DemultiplexingListener{tcpl.(*net.TCPListener), sshh}
	tlsl := tls.NewListener(dl, srv.TLSConfig)
	for {
		log.Printf("Starting server on tcp %v\n", srv.Addr)
		if err = srv.Serve(tlsl); err != nil {
			log.Printf("%v :-(\n", err)
		}
		time.Sleep(50 * time.Millisecond)
		log.Printf("Restarting server\n")
	}
}
