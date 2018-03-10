PrxPass-server
===

PrxPass is a reverse-proxy server that behaves like ngrok, localtunnel, etc.
It's a self-hosted solution, so you need a server.

## Usage

Place `server.pem` and `key.pem` along the `prxpass-server` binary, then:

```
prxpass-server -client 0.0.0.0:8080 -server 0.0.0.0:443
```

This will run a prxpass instance that will accept prxpass-client connections 
at 0.0.0.0:8080 and HTTP connections at https://0.0.0.0/

See [prxpass-client](//github.com/Defman21/prxpass-client) for information about connecting to the server.

