package main

import (
	"log"
	"math/rand"
	"net/http"
	"net/http/cgi"
	"net/http/httputil"
	"path/filepath"
	"strings"

	"github.com/myfreeweb/443d/unixsock"
)

// based on https://stackoverflow.com/questions/26141953/custom-404-with-gorilla-mux-and-std-http-fileserver
type hookedResponseWriter struct {
	http.ResponseWriter
	req    *http.Request
	h      http.Handler
	ignore bool
}

func (hrw *hookedResponseWriter) WriteHeader(status int) {
	if status == 404 {
		hrw.ignore = true
		for k := range hrw.Header() {
			hrw.Header().Del(k)
		}
		hrw.h.ServeHTTP(hrw.ResponseWriter, hrw.req)
	} else {
		hrw.ResponseWriter.WriteHeader(status)
	}
}

func (hrw *hookedResponseWriter) Write(p []byte) (int, error) {
	if hrw.ignore {
		return len(p), nil
	}
	return hrw.ResponseWriter.Write(p)
}

type PathBackend struct {
	Handler http.Handler
	Path    string
	Type    string
	Address string
	Exec    string
	CutPath bool `yaml:"cut_path"`
}

type HttpBackend struct {
	Handler   http.Handler
	Hostnames []string
	Paths     map[string][]PathBackend
}

func proxyHandler(p *PathBackend) http.Handler {
	transp := &http.Transport{MaxIdleConnsPerHost: 100, DisableKeepAlives: true}
	transp.RegisterProtocol("unix", unixsock.NewUnixTransport())
	var h http.Handler
	h = &httputil.ReverseProxy{
		Transport: transp,
		Director: func(r *http.Request) {
			r.URL.Scheme = p.Type
			r.URL.Host = p.Address
			if p.CutPath {
				r.URL.Path = "/" + r.URL.Path // WTF, StripPrefix
			}
		},
	}
	if p.CutPath {
		h = http.StripPrefix(p.Path, h)
	}
	return h
}

func fileHandler(p *PathBackend) http.Handler {
	return http.StripPrefix(p.Path, http.FileServer(http.Dir(p.Address)))
}

func cgiHandler(p *PathBackend) http.Handler {
	fileh := http.FileServer(http.Dir(filepath.Dir(p.Exec)))
	cgih := &cgi.Handler{Path: p.Exec}
	var h http.Handler
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			cgih.ServeHTTP(w, r)
		} else {
			fileh.ServeHTTP(&hookedResponseWriter{
				ResponseWriter: w,
				req:            r,
				h:              cgih,
			}, r)
		}
	})
	if p.CutPath {
		h = http.StripPrefix(p.Path, h)
	}
	return h
}

func (p *PathBackend) Initialize() {
	if p.Type == "" {
		p.Type = "http"
	}
	if p.Type == "unix" || p.Type == "http" {
		p.Handler = proxyHandler(p)
	} else if p.Type == "file" {
		p.Handler = fileHandler(p)
	} else if p.Type == "cgi" {
		p.Handler = cgiHandler(p)
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
