package main

import (
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"

	"github.com/myfreeweb/443d/unixsock"
)

type PathBackend struct {
	Handler http.Handler
	Path    string
	Type    string
	Address string
	CutPath bool `yaml:"cut_path"`
}

type HttpBackend struct {
	Handler   http.Handler
	Hostnames []string
	Paths     map[string][]PathBackend
}

func proxyHandler(backend *PathBackend) http.Handler {
	transp := &http.Transport{MaxIdleConnsPerHost: 100}
	transp.RegisterProtocol("unix", unixsock.NewUnixTransport())
	var h http.Handler
	h = &httputil.ReverseProxy{
		Transport: transp,
		Director: func(r *http.Request) {
			r.URL.Scheme = backend.Type
			r.URL.Host = backend.Address
			if backend.CutPath {
				r.URL.Path = "/" + r.URL.Path // WTF, StripPrefix
			}
		},
	}
	if backend.CutPath {
		h = http.StripPrefix(backend.Path, h)
	}
	return h
}

func (p *PathBackend) Initialize() {
	if p.Type == "" {
		p.Type = "http"
	}
	if p.Type == "unix" || p.Type == "http" {
		p.Handler = proxyHandler(p)
	} else {
		log.Fatalf("Invalid type '%s' for path '%s'", p.Type, p.Path)
	}
}

func backendHandler(b *HttpBackend) http.Handler {
	mux := http.NewServeMux()
	for path := range b.Paths {
		pbackends := b.Paths[path]
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			pbackends[rand.Intn(len(pbackends))].Handler.ServeHTTP(w, r)
		})
	}
	return mux
}

func (b *HttpBackend) Initialize() {
	for path := range b.Paths {
		for pb := range b.Paths[path] {
			b.Paths[path][pb].Path = path
			b.Paths[path][pb].Initialize()
		}
	}
	b.Handler = backendHandler(b)
}
