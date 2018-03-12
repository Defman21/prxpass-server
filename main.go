package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shiena/ansicolor"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

func init() {
	logrus.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true})
}

type client struct {
	conn     net.Conn
	request  chan []byte
	response chan []byte
	close    chan bool
}

func (c *client) writer(id string) {
	msgBytes, err := msgpack.Marshal(&message{
		Sender:  "server",
		Version: 1,
		RPC: rpcCall{
			Method: "net/notify",
			Args:   []string{id},
		},
	})

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
			"id":  id,
		}).Warn("msgpack.Marshal error")
		return
	}

	logrus.WithFields(logrus.Fields{
		"id":     id,
		"method": "net/notify",
		"args":   []string{id},
	}).Info("RPC")
	c.conn.Write(formatMessage(msgBytes))

	for {
		select {
		case httpRequest := <-c.request:
			logrus.WithFields(logrus.Fields{
				"id": id,
			}).Info("HTTP request")
			msgBytes, _ := msgpack.Marshal(&message{
				Sender:  "server",
				Version: 1,
				RPC: rpcCall{
					Method: "tcp/request",
					Args:   []string{string(httpRequest)},
				},
			})
			logrus.WithFields(logrus.Fields{
				"id":     id,
				"method": "tcp/request",
				"args":   []string{"HTTP Dump"},
			}).Info("RPC")
			c.conn.Write(formatMessage(msgBytes))
		case <-c.close:
			logrus.WithFields(logrus.Fields{
				"id": id,
			}).Info("Writter closed")
			return
		}
	}
}

type rpcCall struct {
	Method string
	Args   []string
}

type message struct {
	Sender  string
	Version int
	RPC     rpcCall
}

var clients map[string]*client

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	clients = make(map[string]*client)
}

func formatMessage(msgpackBytes []byte) []byte {
	return append([]byte("!msgpack:"), msgpackBytes...)
}

func parseMessage(msg []byte) (*message, bool, error) {
	if bytes.HasPrefix(msg, []byte("!msgpack:")) {
		var obj message
		err := msgpack.Unmarshal(msg[9:], &obj)
		if err != nil {
			return nil, false, err
		}
		if obj.Version == 0 {
			return nil, false, nil
		}
		return &obj, true, nil
	}
	return nil, false, nil
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
	clientAddr := flag.String("client", ":30303", "Binding address for the client server")
	serverAddr := flag.String("server", ":4444", "Binding address for the http server")
	useHTTPS := flag.Bool("https", false, "Use HTTPS")
	cert := flag.String("cert", "cert.pem", "Path to the cert file")
	key := flag.String("key", "key.pem", "Path to the private key")
	host := flag.String("host", "test.loc", "Hostname of the http server")
	customIDs := flag.Bool("customid", false, "Allow clients to specify custom IDs")

	flag.Parse()

	r := mux.NewRouter()
	s := r.Host(fmt.Sprintf("{subdomain:[a-z0-9]+}.%v", *host)).Subrouter()

	s.HandleFunc("/{url:.*}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if cl, ok := clients[vars["subdomain"]]; ok {
			dump, _ := httputil.DumpRequest(r, true)
			go func() {
				cl.request <- dump
			}()
			for {
				select {
				case data := <-cl.response:
					reader := bufio.NewReader(bytes.NewReader(data))
					resp, _ := http.ReadResponse(reader, r)
					body, _ := ioutil.ReadAll(resp.Body)
					for k := range resp.Header {
						w.Header().Set(k, resp.Header.Get(k))
					}
					w.WriteHeader(resp.StatusCode)
					w.Write(body)
					return
				}
			}
		} else {
			logrus.WithFields(logrus.Fields{
				"id": vars["subdomain"],
			}).Warn("Client not found")
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
			logrus.WithFields(logrus.Fields{
				"con": con,
			}).Info("Client connected")

			if err != nil {
				log.Fatal(err)
			}

			go func(con net.Conn, id string) {
				var cl *client
				logrus.Info("Client goroutine created")
				responseBytes := make([]byte, 1024)
				for {
					n, err := con.Read(responseBytes)
					if err != nil {
						logrus.WithFields(logrus.Fields{
							"id": id,
						}).Info("Client disconnected")
						con.Close()
						if cl != nil {
							cl.close <- true
							delete(clients, id)
						}
						return
					}
					if msgObj, isMsgpack, _ := parseMessage(responseBytes[:n]); isMsgpack {
						switch msgObj.RPC.Method {
						case "net/register":
							logrus.WithFields(logrus.Fields{
								"con":    con,
								"method": "net/register",
								"args":   msgObj.RPC.Args,
							}).Info("RPC")
							cid := msgObj.RPC.Args[0]
							if *customIDs {
								if _, exists := clients[cid]; exists {
									logrus.WithFields(logrus.Fields{
										"id":     id,
										"reason": "in use",
									}).Warn("Custom ID request rejected")
								} else {
									logrus.WithFields(logrus.Fields{
										"oldId": id,
										"newId": cid,
									}).Info("Custom ID request accepted")
									id = cid
								}
							} else {
								logrus.Warn("Custom IDs are disabled")
							}
							clients[id] = &client{
								conn:     con,
								request:  make(chan []byte),
								response: make(chan []byte),
								close:    make(chan bool),
							}
							cl = clients[id]
							logrus.WithFields(logrus.Fields{
								"id": id,
							}).Info("Registered a client")
							go cl.writer(id)
						case "tcp/response":
							logrus.WithFields(logrus.Fields{
								"id":     id,
								"method": "tcp/response",
								"args":   []string{"HTTP Dump"},
							}).Info("RPC")
							response := msgObj.RPC.Args[0]
							cl.response <- []byte(response)
						}
					}
				}
			}(con, id)
		}
	}()
	if *useHTTPS {
		logrus.WithFields(logrus.Fields{
			"https":  *useHTTPS,
			"server": *serverAddr,
			"host":   *host,
			"cert":   *cert,
			"key":    *key,
		}).Info("Listening")
		http.ListenAndServeTLS(*serverAddr, *cert, *key, r)
	} else {
		logrus.WithFields(logrus.Fields{
			"https":  *useHTTPS,
			"server": *serverAddr,
			"host":   *host,
		}).Info("Listening")
		http.ListenAndServe(*serverAddr, r)
	}
}
