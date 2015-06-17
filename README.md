# 443d [![ISC License](https://img.shields.io/badge/license-ISC-red.svg?style=flat)](https://tldrlegal.com/license/-isc-license)

This is

- a reverse HTTP(S) proxy
- written in [Go]
- that proxies to HTTP/1.1 via TCP or UNIX Domain Sockets,
- supports [HTTP/2] over TLS, like [nghttpx]
- and does TLS/SSH demultiplexing, like [sslh].

Basically, it's an [nginx] replacement for websites that don't need advanced load balancing or request modification, but would like good security, easy configuration and deployment (single static binary, thanks to Go).

It's rather small and simple, so it has the following limitations:

- no support for different certificates per domain via SNI (TODO?);
- no websockets (TODO?);
- no configuration reloading;
- no markdown, no built-in git pulling, no basic auth, etc. (by design -- try [Caddy] instead).

Also, it uses Go's TLS library, which currently doesn't support the chacha20-poly1305 ciphersuite.
On the other hand, it's a modern, clean library, not a [pile of legacy C code](https://en.wikipedia.org/wiki/OpenSSL) (not even [a cleaned-up pile of legacy C code](http://www.libressl.org/)).

[Go]: https://golang.org
[HTTP/2]: https://http2.github.io
[nghttpx]: https://nghttp2.org/documentation/nghttpx.1.html
[sslh]: https://github.com/yrutschle/sslh
[nginx]: http://nginx.org 
[Caddy]: https://caddyserver.com

## Installation

Binaries might be available someday.

If you have a Go setup:

```bash
$ go get github.com/myfreeweb/443d
```

## Usage

You need to write a simple configuration file.
The syntax is [YAML].
Here's an example:

```yaml
tls: # 443d will serve TLS there
  listen: 0.0.0.0:443 # IPv6 will magically work too
  ssh: 127.0.0.1:22 # When 443d sees an SSH connection instead of TLS, proxy there
  cert: /etc/certs/server.crt
  key: /etc/certs/server.key
  hsts: # Add the Strict-Transport-Security header
    seconds: 31536000 # max-age
    subdomains: true # includeSubdomains
  hpkp: # Add the Public-Key-Pinning header
    seconds: 5184000 # max-age
    subdomains: true # includeSubdomains
    backup_keys: # You must have at least one hash here!
      - aaaogjIWd0KuaCsQa9Zon7aTON0JapN1fonHra2bdGk=

http: # 443d will serve non-TLS HTTP there
  # (for debugging or to provide access through a Tor hidden service)
  listen: 127.0.0.1:8080

hosts: # 443d will proxy to the following virtual hosts
  - hostnames: ["*.example.com", "example.com", "example.com:*"] # Host header matching
      # (supports glob patterns; if there's a port in the header, it's not removed automatically)
    paths: # URL path prefix matching for this hostname, longer prefixes are matched first
      /git/: # The ending slash is important!!!
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
