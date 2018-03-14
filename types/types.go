package types

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack"
	"net"
)

// RPC an RPC call
type RPC struct {
	Method string
	Args   []string
}

// Message a message
type Message struct {
	Sender  string
	Version int
	RPC
}

// Client a client
type Client struct {
	Conn     net.Conn
	Request  chan *Request
	Response chan *Response
	Close    chan bool
}

// NewClient creates a client struct
func NewClient(con net.Conn) *Client {
	return &Client{
		Conn:     con,
		Request:  make(chan *Request),
		Response: make(chan *Response),
		Close:    make(chan bool),
	}
}

// Clients a list of clients
type Clients map[string]*Client

// Request a request
type Request struct {
	Type string
	Data []byte
}

// Response a response
type Response struct {
	Type string
	Data []byte
}

// Writer a writing goroutine
func (c *Client) Writer(id string) {
	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Info("Writing goroutine created")
	msgBytes, err := CreateMsgpack(&Message{
		Sender:  "server",
		Version: 1,
		RPC: RPC{
			Method: "net/notify",
			Args:   []string{id},
		},
	})

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
			"id":  id,
		}).Warn("helpers.CreateMsgpack error")
		return
	}

	logrus.WithFields(logrus.Fields{
		"id":     id,
		"method": "net/notify",
		"args":   []string{id},
	}).Info("RPC")

	c.Conn.Write(msgBytes)

	for {
		select {
		case reqChan := <-c.Request:
			logrus.WithFields(logrus.Fields{
				"id":   id,
				"type": reqChan.Type,
			}).Info("Request")
			msgBytes, err := CreateMsgpack(&Message{
				Sender:  "server",
				Version: 1,
				RPC: RPC{
					Method: fmt.Sprintf("%v/request", reqChan.Type),
					Args:   []string{string(reqChan.Data)},
				},
			})
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"id":  id,
					"err": err,
				}).Warn("helpers.CreateMsgpack error")
				continue
			}
			logrus.WithFields(logrus.Fields{
				"id":     id,
				"method": fmt.Sprintf("%v/request", reqChan.Type),
			}).Info("RPC")
			c.Conn.Write(msgBytes)
		case <-c.Close:
			logrus.WithFields(logrus.Fields{
				"id":     id,
				"reason": "closed",
			}).Warn("Writing goroutine destroyed")
			return
		}
	}
}

// Reader reading goroutine
func (c *Client) Reader(clients Clients, id string, customIDs bool, password string) {
	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Info("Reading goroutine created")
	responseBytes := make([]byte, 1024)
	for {
		n, err := c.Conn.Read(responseBytes)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"id":     id,
				"reason": err,
			}).Warn("Reading goroutine destroyed")
			c.Conn.Close()
			c.Close <- true
			return
		}
		msgObj, isMsgpack, err := ParseMsgpack(responseBytes[:n])
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"id":  id,
				"err": err,
			}).Warn("ParseMsgpack failed")
			continue
		}
		if isMsgpack {
			switch msgObj.RPC.Method {
			case "net/register":
				logrus.WithFields(logrus.Fields{
					"con":    c.Conn,
					"method": "net/register",
					"args":   msgObj.RPC.Args,
				}).Info("RPC")
				cid := msgObj.RPC.Args[0]
				if password != "" {
					upass := msgObj.RPC.Args[1]
					if upass != password {
						logrus.WithFields(logrus.Fields{
							"password":        password,
							"client_password": upass,
						}).Warn("Password mismatch")
						msgBytes, _ := CreateMsgpack(&Message{
							Sender:  "server",
							Version: 1,
							RPC: RPC{
								Method: "net/auth-reject",
								Args:   []string{"Password mismatch"},
							},
						})
						c.Conn.Write(msgBytes)
						c.Conn.Close()
						logrus.WithFields(logrus.Fields{
							"id":     id,
							"reason": "Password mismatch",
						}).Warn("Writing goroutine destroyed")
						return
					}
				}
				if customIDs {
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
				clients[id] = c
				logrus.WithFields(logrus.Fields{
					"id": id,
				}).Info("Registered a client")
				go c.Writer(id)
			case "tcp/response":
				logrus.WithFields(logrus.Fields{
					"id":     id,
					"method": "tcp/response",
				}).Info("RPC")
				responseStr := msgObj.RPC.Args[0]
				c.Response <- &Response{Type: "tcp", Data: []byte(responseStr)}
			case "http/response":
				logrus.WithFields(logrus.Fields{
					"id":     id,
					"method": "http/response",
				}).Info("RPC")
				responseStr := msgObj.RPC.Args[0]
				c.Response <- &Response{Type: "http", Data: []byte(responseStr)}
			}
		}
	}
}

// CreateMsgpack create a msgpack message
func CreateMsgpack(obj *Message) ([]byte, error) {
	msgpBytes, err := msgpack.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return append([]byte("!msgpack:"), msgpBytes...), nil
}

// ParseMsgpack parse a msgpack message
func ParseMsgpack(msg []byte) (*Message, bool, error) {
	if bytes.HasPrefix(msg, []byte("!msgpack:")) {
		var obj Message
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
