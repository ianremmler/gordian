package gordian

import (
	"code.google.com/p/go.net/websocket"

	"fmt"
	"io"
)

// Control types.
const (
	CONNECT = iota
	REGISTER
	CLOSE
)

// ClientId is a user-defined client identifier, which can be of any hashable type.
type ClientId interface{}

// MessageData is a user-defined message payload.
type MessageData interface{}

// Message is the internal message format
type Message struct {
	From ClientId    // From is the originating client.
	To   ClientId    // To is the destination client.
	Data MessageData // Data is the message payload.
}

// Client stores state and control information for a client.
type Client struct {
	Id      ClientId        // Id is a unique identifier.
	Ctrl    int             // Ctrl is the current control type.
	Conn    *websocket.Conn // Conn is the connection info provided by the websocket package.
	message chan *Message
}

// Gordian processes and distributes messages and manages clients.
type Gordian struct {
	Control chan *Client  // Control is used to pass client control information within Gordian.
	Message chan *Message // Message is used to pass messages within Gordian.
	manage  chan *Client
	clients map[ClientId]*Client
}

// New constructs an initialized Gordian instance.
func New() *Gordian {
	g := &Gordian{
		Control: make(chan *Client),
		Message: make(chan *Message),
		manage:  make(chan *Client),
		clients: make(map[ClientId]*Client),
	}
	return g
}

// Run starts Gordian's event loop.
func (g *Gordian) Run() {
	go func() {
		for {
			select {
			case msg := <-g.Message:
				if client, ok := g.clients[msg.To]; ok {
					client.message <- msg
				}
			case client := <-g.manage:
				switch client.Ctrl {
				case REGISTER:
					g.clients[client.Id] = client
				case CLOSE:
					close(client.message)
					delete(g.clients, client.Id)
				}
			}
		}
	}()
}

// WSHandler returns a websocket.Handler compatible function to handle connections.
func (g *Gordian) WSHandler() func(conn *websocket.Conn) {
	return func(conn *websocket.Conn) {
		g.Control <- &Client{Ctrl: CONNECT, Conn: conn}
		client := <-g.Control
		if client.Id == nil || client.Ctrl != REGISTER {
			return
		}
		client.message = make(chan *Message)
		g.manage <- client
		go g.writeToWS(client)
		g.readFromWS(client)
		client.Ctrl = CLOSE
		g.Control <- client
		g.manage <- client
	}
}

func (g *Gordian) readFromWS(client *Client) {
	for {
		var data MessageData
		err := websocket.JSON.Receive(client.Conn, &data)
		switch err {
		case nil:
			g.Message <- &Message{From: client.Id, Data: data}
		case io.EOF:
			return
		default:
			fmt.Println(err)
		}
	}
}

func (g *Gordian) writeToWS(client *Client) {
	for {
		msg, ok := <-client.message
		if !ok {
			return
		}
		if err := websocket.JSON.Send(client.Conn, msg.Data); err != nil {
			fmt.Println(err)
		}
	}
}
