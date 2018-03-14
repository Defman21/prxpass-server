package main

import (
	"flag"
	handlerHTTP "github.com/Defman21/prxpass-server/handlers/http"
	"github.com/Defman21/prxpass-server/helpers"
	"github.com/Defman21/prxpass-server/types"
	"github.com/shiena/ansicolor"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"os"
	"time"
)

func init() {
	logrus.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true})
}

var clients types.Clients

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	clients = make(types.Clients)
}

func main() {
	clientAddr := flag.String("client", ":30303", "Binding address for the client server")
	serverAddr := flag.String("server", ":4444", "Binding address for the http server")
	useHTTPS := flag.Bool("https", false, "Use HTTPS")
	cert := flag.String("cert", "cert.pem", "Path to the cert file")
	key := flag.String("key", "key.pem", "Path to the private key")
	host := flag.String("host", "test.loc", "Hostname of the http server")
	password := flag.String("password", "", "Require the password from clients")
	customIDs := flag.Bool("customid", false, "Allow clients to specify custom IDs")

	flag.Parse()

	ln, err := net.Listen("tcp", *clientAddr)

	if err != nil {
		logrus.Fatal(err)
	}

	go func() {
		for {
			con, err := ln.Accept()
			id := helpers.ID()
			logrus.WithFields(logrus.Fields{
				"con": con,
			}).Info("Client connected")

			if err != nil {
				logrus.Fatal(err)
			}

			cl := types.NewClient(con)
			go cl.Reader(clients, id, *customIDs, *password)
		}
	}()

	handlerHTTP.Handle(clients, *useHTTPS, *serverAddr, *host, *cert, *key)
}
