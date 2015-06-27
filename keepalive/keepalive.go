package keepalive

import (
	"net"
	"time"
)

// https://golang.org/src/net/http/server.go#L1971

type KeepAliveListener struct {
	*net.TCPListener
}

func (kal KeepAliveListener) Accept() (net.Conn, error) {
	tc, err := kal.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(2 * time.Minute)
	return tc, nil
}
