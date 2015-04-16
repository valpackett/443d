package main

import (
	"crypto/tls"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/bradfitz/http2"
	"github.com/myfreeweb/443d/demux"
	"github.com/myfreeweb/443d/unixsock"
	"github.com/myfreeweb/443d/util"
	"github.com/naoina/toml"
	"github.com/ryanuber/go-glob"
)

type HttpBackend struct {
	Paths map[string][]struct {
		Net     string
		Address string
		CutPath bool
	}
	PathOrder []string
}

type Config struct {
	Listen string
	Cert   string
	Key    string
	Http   map[string]HttpBackend
	Ssh    struct {
		Address string
	}
}

var confpath = flag.String("config", "/usr/local/etc/443d.toml", "path to the configuration file")
var config Config

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	readConfig()
	sshh := demux.SshHandler(config.Ssh.Address)
	srv := &http.Server{
		Addr:    config.Listen,
		Handler: httpHandler(),
	}
	http2.ConfigureServer(srv, &http2.Server{})
	srv.TLSConfig.Certificates = make([]tls.Certificate, 1)
	srv.TLSConfig.Certificates[0], _ = tls.LoadX509KeyPair(config.Cert, config.Key)
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

func httpHandler() *httputil.ReverseProxy {
	transp := &http.Transport{MaxIdleConnsPerHost: 100}
	transp.RegisterProtocol("unix", unixsock.NewUnixTransport())
	return &httputil.ReverseProxy{
		Transport: transp,
		Director: func(r *http.Request) {
			for hostn, hostcnf := range config.Http {
				if hostn != "*" && glob.Glob(hostn, r.URL.Host) {
					applyHost(&hostcnf, r)
					return
				}
			}
			defhost := config.Http["*"] // TODO: check
			applyHost(&defhost, r)
		},
	}
}

func applyHost(hostcnf *HttpBackend, r *http.Request) {
	for _, pathprefix := range hostcnf.PathOrder {
		if strings.HasPrefix(r.URL.Path, pathprefix) {
			backends := hostcnf.Paths[pathprefix]
			backend := backends[rand.Intn(len(backends))]
			if backend.Net == "" {
				r.URL.Scheme = "http"
			} else {
				r.URL.Scheme = backend.Net
			}
			r.URL.Host = backend.Address
			if backend.CutPath {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, pathprefix)
			}
			return
		}
	}
}

func readConfig() {
	flag.Parse()
	f, err := os.Open(*confpath)
	if err != nil {
		log.Fatalf("%v :-(\n", err)
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("%v :-(\n", err)
	}
	if err := toml.Unmarshal(buf, &config); err != nil {
		log.Fatalf("%v :-(\n", err)
	}
	for ib := range config.Http {
		var order []string
		for path := range config.Http[ib].Paths {
			order = append(order, path)
		}
		sort.Sort(util.ByLengthDesc(order))
		config.Http[ib] = HttpBackend{Paths: config.Http[ib].Paths, PathOrder: order}
	}
}
