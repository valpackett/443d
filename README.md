# 443d [![ISC License](https://img.shields.io/badge/license-ISC-red.svg?style=flat)](https://tldrlegal.com/license/-isc-license)

A simple reverse proxy with [HTTP/2] support & TLS/SSH demultiplexing.

Basically, like [nghttpx] + [sslh], but in [Go].

[HTTP/2]: https://http2.github.io
[nghttpx]: https://nghttp2.org/documentation/nghttpx.1.html
[sslh]: https://github.com/yrutschle/sslh
[Go]: https://golang.org

## Installation

Binaries will be available soon.

If you have a Go setup:

```bash
$ go get github.com/myfreeweb/443d
```

## Usage

```bash
$ 443d -http="localhost:8080" -ssh="localhost:22" -listen="0.0.0.0:443" -crt="server.crt" -key="server.key"
```

Use [supervisord] or something like that to run in production, it does not daemonize itself & logs to stderr.

Do not run as root, instead...

- FreeBSD: remove the whole port number restriction: `sysctl net.inet.ip.portrange.reservedhigh=0 && echo "net.inet.ip.portrange.reservedhigh=0" >> /etc/sysctl.conf`
- Linux: allow it to bind to low port numbers: `setcap 'cap_net_bind_service=+ep' $(which 443d)`
- anywhere: run on a different port and redirect ports in the firewall

You can make a chroot for it easily, you only need `/dev/urandom`.

[supervisord]: http://supervisord.org

## License

Copyright 2015 Greg V <greg@unrelenting.technology>  
Available under the ISC license, see the `COPYING` file
