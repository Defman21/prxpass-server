package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
)

type client struct {
	conn     net.Conn
	request  chan []byte
	response chan []byte
	channel  chan bool
}

var clients map[string]*client

func init() {
	clients = make(map[string]*client)
}

func randStr() string {
	letter := []rune("abcdefghijklmnopqrstuvwxyz1234567890")

	b := make([]rune, 20)

	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}

	return string(b)
}

func main() {
	clientAddr := flag.String("client", ":30303", "Client address")
	serverAddr := flag.String("server", ":4444", "Server address")
	useHTTPS := flag.Bool("https", false, "Use HTTPS")
	cert := flag.String("cert", "cert.pem", "Path to the cert file (https = true)")
	key := flag.String("key", "key.pem", "Path to the private key (https = true)")
	host := flag.String("host", "test.loc", "Hostname for the clients")
	flag.Parse()
	r := mux.NewRouter()
	s := r.Host(fmt.Sprintf("{subdomain:[a-z0-9]+}.%v", *host)).Subrouter()
	s.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if cl, ok := clients[vars["subdomain"]]; ok {
			dump, _ := httputil.DumpRequest(r, true)
			go func() {
				log.Println("Sending the request to the %v channel", vars["subdomain"])
				cl.request <- dump
			}()
			for {
				select {
				case data := <-cl.response:
					reader := bufio.NewReader(bytes.NewReader(data))
					resp, _ := http.ReadResponse(reader, r)
					body, _ := ioutil.ReadAll(resp.Body)
					for k, _ := range resp.Header {
						w.Header().Set(k, resp.Header.Get(k))
					}
					w.Write(body)
					return
				}
			}
		} else {
			log.Printf("Client %v not found", vars["subdomain"])
			w.Write([]byte("Client not found"))
			return
		}
	})

	ln, err := net.Listen("tcp", *clientAddr)

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			con, err := ln.Accept()
			id := randStr()
			log.Printf("A client connected: %v = %+v", id, con)

			clients[id] = &client{
				conn:     con,
				request:  make(chan []byte),
				response: make(chan []byte),
				channel:  make(chan bool),
			}

			if err != nil {
				log.Fatal(err)
			}

			go func(cl *client, id string) {
				log.Println("Reading goroutine created")
				responseBytes := make([]byte, 1024)
				for {
					n, err := cl.conn.Read(responseBytes)
					if err != nil {
						log.Printf("The client %v disconnected", id)
						cl.conn.Close()
						cl.channel <- true
						delete(clients, id)
						return
					} else {
						responseBytes := responseBytes[:n]
						log.Printf("A client sent a reponse:\n%v", string(responseBytes))
						cl.response <- responseBytes
					}
				}
			}(clients[id], id)

			go func(cl *client, id string) {
				log.Println("Writing goroutine created")
				cl.conn.Write([]byte(fmt.Sprintf("~!@=%v=@!~", id)))
				for {
					select {
					case data := <-cl.request:
						log.Printf("Incoming request:\n%s", string(data))
						con.Write(data)
					case <-cl.channel:
						log.Printf("Stopped by the %v channel", id)
						return
					}
				}
			}(clients[id], id)
		}
	}()
	if *useHTTPS {
		log.Printf("Listening at: https://%v (%v)", *host, *serverAddr)
		http.ListenAndServeTLS(*serverAddr, *cert, *key, r)
	} else {
		log.Printf("Listening at: http://%v (%v)", *host, *serverAddr)
		http.ListenAndServe(*serverAddr, r)
	}
}
