package main

import (
    "log"
    "net"
    "net/http"
    "net/http/httputil"
    "bytes"
    "io/ioutil"
    "bufio"
    "github.com/gorilla/mux"
    "flag"
)

func main() {
    clientAddr := flag.String("client", ":30303", "Client address")
    serverAddr := flag.String("server", ":4444", "Server address")
    flag.Parse()
    r := mux.NewRouter()
    request  := make(chan []byte)
    response := make(chan []byte)
    r.HandleFunc("/{msg}", func (w http.ResponseWriter, r *http.Request) {
        dump, _ := httputil.DumpRequest(r, true)
        go func() {
            log.Println("Sending the request to the channel")
            request <- dump
        }()
        for {
            select {
            case data := <-response:
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
    })

    ln, err := net.Listen("tcp", *clientAddr)

    if err != nil {
        log.Fatal(err)
    }

    go func() {
        for {
            closeWriteChannel := make(chan bool)
            con, err := ln.Accept()
            log.Printf("A client connected: %+v", con)

            if err != nil {
                log.Fatal(err)
            }

            go func() {
                log.Println("Reading goroutine created")
                responseBytes := make([]byte, 1024)
                for {
                    n, err := con.Read(responseBytes)
                    if err != nil {
                        log.Printf("A client disconnected")
                        con.Close()
                        closeWriteChannel <- true
                        return
                    } else {
                        responseBytes := responseBytes[:n]
                        log.Printf("A client sent a reponse:\n%v", string(responseBytes))
                        response <- responseBytes
                    }
                }
            }()

            go func() {
                log.Println("Writing goroutine created")
                for {
                    select {
                    case data := <-request:
                        log.Printf("Incoming request:\n%s", string(data))
                        con.Write(data)
                    case <-closeWriteChannel:
                        log.Printf("Stopped by a channel")
                        return
                    }
                }
            }()
        }
    }()
    http.ListenAndServeTLS(*serverAddr, "server.pem", "key.pem", r)
}

