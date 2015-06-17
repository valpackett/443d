package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
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
	"github.com/ryanuber/go-glob"
	"gopkg.in/yaml.v2"
)

type HttpBackend struct {
	Hostnames []string
	Paths     map[string][]struct {
		Type    string
		Address string
		CutPath bool `yaml:"cut_path"`
	}
	PathOrder []string
}

type Config struct {
	Tls struct {
		Listen string
		Cert   string
		Key    string
		Ssh    string
		Hsts   struct {
			Seconds    int
			Subdomains bool
		}
		Hpkp struct {
			Seconds    int
			Subdomains bool
			BackupKeys []string `yaml:"backup_keys"`
		}
	}
	Http struct {
		Listen string
	}
	Hosts       []HttpBackend
	DefaultHost string
}

var confpath = flag.String("config", "/usr/local/etc/443d.yaml", "path to the configuration file")
var config Config
var tlsKeyPair tls.Certificate
var hstsHeader string
var hpkpHeader string

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	readConfig()
	handler := httpHandler()
	go func() {
		addr := config.Http.Listen
		if addr == "" {
			log.Printf("No listen address for the HTTP server \n")
			return
		}
		srv := &http.Server{Addr: addr, Handler: handler}
		tcpl, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("%v :-(\n", err)
		}
		serve("HTTP server", srv, tcpl)
	}()
	go func() {
		addr := config.Tls.Listen
		if addr == "" {
			log.Printf("No listen address for the TLS server \n")
			return
		}
		if config.Tls.Cert == "" && config.Tls.Key == "" {
			log.Printf("No keypair for the TLS server \n")
			return
		}
		secHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.Tls.Hsts.Seconds != 0 {
				w.Header().Add("Strict-Transport-Security", hstsHeader)
			}
			if config.Tls.Hpkp.Seconds != 0 {
				w.Header().Add("Public-Key-Pins", hpkpHeader)
			}
			handler.ServeHTTP(w, r)
		})
		srv := &http.Server{Addr: addr, Handler: secHandler}
		http2.ConfigureServer(srv, &http2.Server{})
		srv.TLSConfig.Certificates = make([]tls.Certificate, 1)
		srv.TLSConfig.Certificates[0] = tlsKeyPair
		tcpl, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("%v :-(\n", err)
		}
		sshh := demux.SshHandler(config.Tls.Ssh)
		dl := &demux.DemultiplexingListener{tcpl.(*net.TCPListener), sshh}
		tlsl := tls.NewListener(dl, srv.TLSConfig)
		serve("TLS server", srv, tlsl)
	}()
	for {
		time.Sleep(500 * time.Millisecond)
	}
}

func serve(name string, srv *http.Server, listener net.Listener) {
	for {
		log.Printf("Starting the "+name+" on tcp %v\n", srv.Addr)
		if err := srv.Serve(listener); err != nil {
			log.Printf(name+" error: %v :-(\n", err)
		}
		time.Sleep(200 * time.Millisecond)
		log.Printf("Restarting the " + name + "\n")
	}
}

func httpHandler() *httputil.ReverseProxy {
	transp := &http.Transport{MaxIdleConnsPerHost: 100}
	transp.RegisterProtocol("unix", unixsock.NewUnixTransport())
	return &httputil.ReverseProxy{
		Transport: transp,
		Director: func(r *http.Request) {
			if r.Host == "" {
				r.Host = config.DefaultHost
			}
			for hostid := range config.Hosts {
				hostcnf := config.Hosts[hostid]
				for hostnid := range hostcnf.Hostnames {
					hostn := hostcnf.Hostnames[hostnid]
					if glob.Glob(hostn, r.Host) {
						applyHost(&hostcnf, r)
						return
					}
				}
			}
		},
	}
}

func applyHost(hostcnf *HttpBackend, r *http.Request) {
	for _, pathprefix := range hostcnf.PathOrder {
		if strings.HasPrefix(r.URL.Path, pathprefix) {
			backends := hostcnf.Paths[pathprefix]
			backend := backends[rand.Intn(len(backends))]
			if backend.Type == "" {
				r.URL.Scheme = "http"
			} else {
				r.URL.Scheme = backend.Type
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
	if err := yaml.Unmarshal(buf, &config); err != nil {
		log.Fatalf("%v :-(\n", err)
	}
	for ib := range config.Hosts {
		var order []string
		for path := range config.Hosts[ib].Paths {
			order = append(order, path)
		}
		sort.Sort(util.ByLengthDesc(order))
		config.Hosts[ib].PathOrder = order
	}
	if config.DefaultHost == "" {
		config.DefaultHost = "localhost"
	}
	if config.Tls.Cert != "" && config.Tls.Key != "" {
		tlsKeyPair, err = tls.LoadX509KeyPair(config.Tls.Cert, config.Tls.Key)
		if err != nil {
			log.Fatalf("Error reading TLS key/cert: %v :-(", err)
		}
		tlsKeyPair.Leaf, err = x509.ParseCertificate(tlsKeyPair.Certificate[len(tlsKeyPair.Certificate)-1])
		if err != nil {
			log.Fatalf("Error parsing TLS cert: %v :-(", err)
		}
		if config.Tls.Hsts.Seconds != 0 {
			hstsHeader = fmt.Sprintf("max-age=%d", config.Tls.Hsts.Seconds)
			if config.Tls.Hsts.Subdomains {
				hstsHeader += "; includeSubdomains"
			}
		}
		if config.Tls.Hpkp.Seconds != 0 {
			if len(config.Tls.Hpkp.BackupKeys) < 1 {
				log.Printf("You should add a backup key to HPKP backup_keys!\n")
			}
			hash := sha256.Sum256(tlsKeyPair.Leaf.RawSubjectPublicKeyInfo)
			hpkpHeader = fmt.Sprintf("pin-sha256=\"%s\"", base64.StdEncoding.EncodeToString(hash[0:]))
			for k := range config.Tls.Hpkp.BackupKeys {
				hpkpHeader += fmt.Sprintf("; pin-sha256=\"%s\"", config.Tls.Hpkp.BackupKeys[k])
			}
			hpkpHeader += fmt.Sprintf("; max-age=%d", config.Tls.Hpkp.Seconds)
			if config.Tls.Hpkp.Subdomains {
				hpkpHeader += "; includeSubdomains"
			}
		}
	}
}
