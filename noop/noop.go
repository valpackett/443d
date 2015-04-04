package noop

import (
	"net"
	"time"
)

type Adr struct{}

func (Adr) Network() string { return "noop" }
func (Adr) String() string  { return "127.0.0.1" }

type Conn struct{}

func (Conn) Read(p []byte) (int, error)         { return 0, nil }
func (Conn) Write(p []byte) (int, error)        { return 0, nil }
func (Conn) Close() error                       { return nil }
func (Conn) LocalAddr() net.Addr                { return Adr{} }
func (Conn) RemoteAddr() net.Addr               { return Adr{} }
func (Conn) SetDeadline(t time.Time) error      { return nil }
func (Conn) SetReadDeadline(t time.Time) error  { return nil }
func (Conn) SetWriteDeadline(t time.Time) error { return nil }
