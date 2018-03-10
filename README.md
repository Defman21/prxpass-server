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

## Options

* `-client <string>` - Address for the client connection handler. (e.g. `0.0.0.0:8080`)
* `-server <string>` - Address of the web-server. (e.g. `0.0.0.0:80` or `0.0.0.0:443`)
* `-host <string>` - Hostname of your server. (e.g. `defman.me`)
* `-https <bool>` - Enable or disable HTTPS.
* `-cert <string>` - Path to the certificate file. (e.g. `./cert.pem`)
* `-key <string>` - Path to the private key file. (e.g. `./key.pem`)
* `-customid <bool>` - Allow clients to specify their IDs instead of the generated ones.

See [prxpass-client](//github.com/Defman21/prxpass-client) for information about connecting to the server.

