package unixsock

import (
	"net"
	"net/http"
)

type UnixTransport struct {
	ht *http.Transport
}

func (ut *UnixTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	req.URL.Scheme = "http"
	return ut.ht.RoundTrip(req)
}

func NewUnixTransport() *UnixTransport {
	return &UnixTransport{
		ht: &http.Transport{
			Dial: func(network, address string) (net.Conn, error) {
				address = address[:len(address)-3] // remove :80
				return net.Dial("unix", address)
			},
		},
	}
}
