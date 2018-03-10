PrxPass-server
===

PrxPass is a reverse-proxy server that behaves like ngrok, localtunnel, etc.
It's a self-hosted solution, so you need a server.

## Usage

```
prxpass-server -client 0.0.0.0:8080 -server 0.0.0.0:80 -host mydomain.me
```

This will run a prxpass instance that will accept prxpass-client connections 
at 0.0.0.0:8080 and HTTP connections at http://0.0.0.0/

### HTTPS

```
prxpass-server -client 0.0.0.0:8080 -server 0.0.0.0:443 -https -cert cert.pem -key key.pem -host mydomain.me
```

See [prxpass-client](//github.com/Defman21/prxpass-client) for information about connecting to the server.

