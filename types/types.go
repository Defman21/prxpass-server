package types

import (
	"bytes"
	"fmt"
	"net"

	"github.com/Defman21/prxpass-server/common"
	"github.com/vmihailenco/msgpack"
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
func (c *Client) Writer(id string, config *HTTPConfig) {
	common.Logger.Infow("Writing goroutine created",
		"id", id,
	)
	url := fmt.Sprintf("http://%s.%s:%d/", id, config.Host, config.ServerPort)
	msgBytes, err := NewMessage(&Message{
		Sender:  "server",
		Version: 1,
		RPC: RPC{
			Method: "net/notify",
			Args:   []string{id, url},
		},
	})

	if err != nil {
		common.Logger.Warnw("helpers.NewMessage error",
			"err", err,
			"id", id,
		)
		return
	}

	common.Logger.Infow("RPC",
		"id", id,
		"method", "net/notify",
		"args", []string{id, url},
	)

	c.Conn.Write(msgBytes)

	for {
		select {
		case reqChan := <-c.Request:
			common.Logger.Infow("Info",
				"id", id,
				"type", reqChan.Type,
			)
			msgBytes, err := NewMessage(&Message{
				Sender:  "server",
				Version: 1,
				RPC: RPC{
					Method: fmt.Sprintf("%v/request", reqChan.Type),
					Args:   []string{string(reqChan.Data)},
				},
			})
			if err != nil {
				common.Logger.Warnw("helpers.NewMessage error",
					"id", id,
					"err", err,
				)
				continue
			}
			common.Logger.Infow("RPC",
				"id", id,
				"method", fmt.Sprintf("%v/request", reqChan.Type),
			)
			c.Conn.Write(msgBytes)
		case <-c.Close:
			common.Logger.Warnw("Writing goroutine destroyed",
				"id", id,
				"reason", "closed",
			)
			return
		}
	}
}

// Reader reading goroutine
func (c *Client) Reader(clients *Clients, id string, config *HTTPConfig) {
	password := config.Password
	customIDs := config.CustomIDs
	common.Logger.Infow("Reading goroutine created",
		"id", id,
	)
	responseBytes := make([]byte, 1024)
	for {
		n, err := c.Conn.Read(responseBytes)
		if err != nil {
			common.Logger.Warnw("Reading goroutine destroyed",
				"id", id,
				"reason", err,
			)
			c.Conn.Close()
			delete(*clients, id)
			c.Close <- true
			return
		}
		msgObj, isMsgpack, err := ParseMessage(responseBytes[:n])
		if err != nil {
			common.Logger.Warnw("ParseMessage failed",
				"id", id,
				"err", err,
			)
			continue
		}
		if isMsgpack {
			switch msgObj.RPC.Method {
			case "net/register":
				common.Logger.Warnw("RPC",
					"con", c.Conn,
					"method", "net/register",
					"args", msgObj.RPC.Args,
				)
				cid := msgObj.RPC.Args[0]
				if password != "" {
					upass := msgObj.RPC.Args[1]
					if upass != password {
						common.Logger.Warnw("Password mismatch",
							"password", password,
							"client_password", upass,
						)
						msgBytes, _ := NewMessage(&Message{
							Sender:  "server",
							Version: 1,
							RPC: RPC{
								Method: "net/auth-reject",
								Args:   []string{"Password mismatch"},
							},
						})
						c.Conn.Write(msgBytes)
						c.Conn.Close()
						common.Logger.Warnw("Writing goroutine destroyed",
							"id", id,
							"reason", "Password mismatch",
						)
						return
					}
				}
				if customIDs {
					if _, exists := (*clients)[cid]; exists {
						common.Logger.Warnw("Custom ID request rejected",
							"id", id,
							"reason", "in use",
						)
					} else {
						common.Logger.Infow("Custom ID request accepted",
							"oldId", id,
							"newId", cid,
						)
						id = cid
					}
				} else {
					common.Logger.Warn("Custom IDs are disabled")
				}
				(*clients)[id] = c
				common.Logger.Infow("Registered a client",
					"id", id,
				)
				go c.Writer(id, config)
			case "tcp/response":
				common.Logger.Infow("RPC",
					"id", id,
					"method", "tcp/response",
				)
				responseStr := msgObj.RPC.Args[0]
				c.Response <- &Response{Type: "tcp", Data: []byte(responseStr)}
			case "http/response":
				common.Logger.Infow("RPC",
					"id", id,
					"method", "http/response",
				)
				responseStr := msgObj.RPC.Args[0]
				c.Response <- &Response{Type: "http", Data: []byte(responseStr)}
			}
		}
	}
}

// NewMessage create a msgpack message
func NewMessage(obj *Message) ([]byte, error) {
	msgpBytes, err := msgpack.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return append([]byte("!msgpack:"), msgpBytes...), nil
}

// ParseMessage parse a msgpack message
func ParseMessage(msg []byte) (*Message, bool, error) {
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
