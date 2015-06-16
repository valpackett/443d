# 443d [![ISC License](https://img.shields.io/badge/license-ISC-red.svg?style=flat)](https://tldrlegal.com/license/-isc-license)

A reverse proxy with [HTTP/2] support & TLS/SSH demultiplexing.

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

You need to write a simple configuration file.
The syntax is [YAML].
Here's an example:

```yaml
# This is an example configuration for 443d.

tls: # 443d will serve TLS there
  listen: 0.0.0.0:443 # IPv6 will magically work too
  cert: /etc/certs/server.crt
  key: /etc/certs/server.key
  ssh: 127.0.0.1:22 # When 443d sees an SSH connection instead of TLS, forward there

http: # 443d will serve non-TLS HTTP there (for debugging or to provide access through a Tor hidden service)
  listen: 127.0.0.1:8080

hosts: # 443d will proxy to the following virtual hosts
  - hostnames: ["*.example.com", "example.com", "example.com:*"] # Host header matching
      # (supports glob patterns; if there's a port in the header, it's not removed automatically)
    paths: # URL path prefix matching for this hostname, longer prefixes are matched first
      /git:
        - type: unix # default is http
          address: /var/run/gitweb/gitweb.sock # format depends on the type
          cut_path: true # means the backend will see /git as /, /git/path as /path, etc. default is false
      /:
        # You can have multiple backends, requests will be load-balanced randomly
        - address: localhost:8080
        - address: localhost:8081
        - address: localhost:8082

defaulthost: example.com # Where to proxy when no Host header is sent
```

Now run the binary:

```bash
$ 443d -config="/usr/local/etc/443d.yaml"
```

Use [supervisord] or something like that to run in production, it does not daemonize itself & logs to stderr.

Do not run as root, instead...

- FreeBSD: remove the whole port number restriction: `sysctl net.inet.ip.portrange.reservedhigh=0 && echo "net.inet.ip.portrange.reservedhigh=0" >> /etc/sysctl.conf`
- Linux: allow it to bind to low port numbers: `setcap 'cap_net_bind_service=+ep' $(which 443d)`
- anywhere: run on a different port and redirect ports in the firewall

You can make a chroot for it easily, you only need `/dev/urandom`.

[YAML]: http://yaml.org
[supervisord]: http://supervisord.org

## License

Copyright 2015 Greg V <greg@unrelenting.technology>  
Available under the ISC license, see the `COPYING` file
