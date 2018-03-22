package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
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

type httpConfig struct {
	Client    string
	Server    string
	Host      string
	CustomIDs bool `toml:"custom_ids"`
	TLS       httpTLSConfig
	Password  string
}

type httpTLSConfig struct {
	Enabled bool
	Cert    string
	Key     string
}

type tcpConfig struct {
	Client   string
	Server   string
	Password string
}

type config struct {
	HTTP httpConfig `toml:"http"`
	TCP  tcpConfig  `toml:"tcp"`
}

func init() {
	logrus.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true})
}

var conf config

func init() {
	if _, err := toml.DecodeFile("./config.toml", &conf); err != nil {
		logrus.Fatal(err)
	}
	logrus.WithFields(logrus.Fields{
		"conf": fmt.Sprintf("%+v", conf),
	}).Info("Config")
}

var clients types.Clients

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	clients = make(types.Clients)
}

func main() {
	isHTTP := flag.Bool("http", true, "Use HTTP")
	isTCP := flag.Bool("tcp", false, "Use TCP")

	var (
		clientAddr string
	)

	if *isHTTP {
		clientAddr = conf.HTTP.Client
	} else if *isTCP {
		clientAddr = conf.TCP.Client
	}

	flag.Parse()
	ln, err := net.Listen("tcp", clientAddr)

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
			go cl.Reader(clients, id, conf.HTTP.CustomIDs, conf.HTTP.Password)
		}
	}()

	handlerHTTP.Handle(clients, conf.HTTP.TLS.Enabled, conf.HTTP.Server, conf.HTTP.Host, conf.HTTP.TLS.Cert, conf.HTTP.TLS.Key)
}
