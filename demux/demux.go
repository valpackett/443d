package demux

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"time"

	"github.com/myfreeweb/443d/noop"
)

type BufConn struct {
	r *bufio.Reader
	net.Conn
}

func (wc *BufConn) Peek(n int) ([]byte, error) { return wc.r.Peek(n) }
func (wc *BufConn) Read(p []byte) (int, error) { return wc.r.Read(p) }

func NewBufConn(c net.Conn) *BufConn {
	return &BufConn{bufio.NewReader(c), c}
}

type DemultiplexingListener struct {
	Listener   *net.TCPListener
	SshHandler func(net.Conn)
}

func (dl *DemultiplexingListener) Accept() (net.Conn, error) {
	c, err := dl.Listener.AcceptTCP()
	if err != nil {
		return nil, err
	}
	c.SetKeepAlive(true)
	c.SetKeepAlivePeriod(2 * time.Minute)
	wc := NewBufConn(c)
	bs, err := wc.Peek(4)
	if err != nil {
		return nil, err
	}
	// RFC 4253: When the connection has been established, both sides MUST send an identification string.
	if bytes.Equal(bs, []byte{83, 83, 72, 45}) { // "SSH-"
		log.Printf("Accepted SSH connection!!! Remote IP: %s\n", wc.RemoteAddr())
		go dl.SshHandler(wc)
		return noop.Conn{}, nil
	} else {
		// log.Printf("Accepted HTTP connection.\n")
		return wc, nil
	}
}

func (dl *DemultiplexingListener) Close() error   { return dl.Listener.Close() }
func (dl *DemultiplexingListener) Addr() net.Addr { return dl.Listener.Addr() }

func SshHandler(host string) func(inc net.Conn) {
	return func(inc net.Conn) {
		outc, err := net.Dial("tcp", host)
		if err != nil {
			log.Printf("SSH dialing failed: %v\n", err)
			inc.Close()
			return
		}
		go io.Copy(inc, outc)
		go io.Copy(outc, inc)
	}
}
