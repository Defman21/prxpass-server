package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Defman21/prxpass-server/common"
	handlerHTTP "github.com/Defman21/prxpass-server/handlers/http"
	"github.com/Defman21/prxpass-server/helpers"
	"github.com/Defman21/prxpass-server/types"
)

var conf types.Config

func init() {
	if _, err := toml.DecodeFile("./config.toml", &conf); err != nil {
		common.Logger.Fatal(err)
	}
	common.Logger.Infow("Config",
		"config", fmt.Sprintf("%+v", conf),
	)
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
		serverAddress string
		clientAddress string
	)

	if *isHTTP {
		clientAddr := conf.HTTP.ClientAddr
		clientPort := conf.HTTP.ClientPort
		clientAddress = fmt.Sprintf("%s:%d", clientAddr, clientPort)

	} else if *isTCP {
		clientAddress = conf.TCP.Client
	}

	flag.Parse()
	ln, err := net.Listen("tcp", clientAddress)
	common.Logger.Infow("Listening [clients]",
		"address", clientAddress,
	)

	if err != nil {
		common.Logger.Fatal(err)
	}

	go func() {
		for {
			con, err := ln.Accept()
			id := helpers.ID()
			common.Logger.Infow("Client connected",
				"con", con,
			)

			if err != nil {
				common.Logger.Fatal(err)
			}

			cl := types.NewClient(con)
			go cl.Reader(&clients, id, &conf.HTTP)
		}
	}()

	serverAddress = fmt.Sprintf("%s:%d", conf.HTTP.ServerAddr, conf.HTTP.ServerPort)

	handlerHTTP.Handle(&clients, conf.HTTP.TLS.Enabled, serverAddress, conf.HTTP.Host, conf.HTTP.TLS.Cert, conf.HTTP.TLS.Key)
}
