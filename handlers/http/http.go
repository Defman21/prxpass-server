package http

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/Defman21/prxpass-server/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
)

// Handle http client handler
func Handle(clients types.Clients, useHTTPS bool, serverAddr, host, cert, key string) {
	r := mux.NewRouter()
	s := r.Host(fmt.Sprintf("{subdomain:[a-z0-9]+}.%v", host)).Subrouter()

	s.HandleFunc("/{url:.*}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if cl, ok := clients[vars["subdomain"]]; ok {
			dump, _ := httputil.DumpRequest(r, true)
			go func() {
				cl.Request <- &types.Request{Type: "http", Data: dump}
			}()
			for {
				select {
				case respChan := <-cl.Response:
					if respChan.Type != "http" {
						logrus.WithFields(logrus.Fields{
							"type": respChan.Type,
						}).Warn("HTTP: Unsupported response type")
						w.Write([]byte("Unsupported response type"))
					}
					reader := bufio.NewReader(bytes.NewReader(respChan.Data))
					resp, _ := http.ReadResponse(reader, r)
					defer func() {
						if err := recover(); err != nil {
							w.Write([]byte(fmt.Sprintf("Panic!\n%v\n%v", string(dump), string(respChan.Data))))
							return
						}
					}()
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
	if useHTTPS {
		logrus.WithFields(logrus.Fields{
			"https":  useHTTPS,
			"server": serverAddr,
			"host":   host,
			"cert":   cert,
			"key":    key,
		}).Info("Listening")
		http.ListenAndServeTLS(serverAddr, cert, key, r)
	} else {
		logrus.WithFields(logrus.Fields{
			"https":  useHTTPS,
			"server": serverAddr,
			"host":   host,
		}).Info("Listening")
		http.ListenAndServe(serverAddr, r)
	}
}
