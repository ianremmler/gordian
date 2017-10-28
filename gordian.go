package gordian

import (
	"github.com/gorilla/websocket"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Control types.
const (
	Connect = iota
	Register
	Establish
	Abort
	Close
)

var (
	upgrader = websocket.Upgrader{}
)

// ClientId is a user-defined client identifier, which can be of any hashable type.
type ClientId interface{}

// MessageData is a user-defined message payload.
type MessageData interface{}

// Message is the internal message format
type Message struct {
	From ClientId    // From is the originating client.
	To   ClientId    // To is the destination client.
	Type string      // Type is the type of message.
	Data MessageData // Data is the message payload.
}

// Unmarshal decodes json data in an incoming message
func (m *Message) Unmarshal(data interface{}) error {
	jsonData, ok := m.Data.(json.RawMessage)
	if !ok {
		return errors.New("Data is not a json.RawMessage")
	}
	return json.Unmarshal(jsonData, data)
}

// Client stores state and control information for a client.
type Client struct {
	Id      ClientId        // Id is a unique identifier.
	Ctrl    int             // Ctrl is the current control type.
	Conn    *websocket.Conn // Conn is the connection info provided by the websocket package.
	Request *http.Request   // Request is the original http request
	outBox  chan Message
}

// Gordian processes and distributes messages and manages clients.
type Gordian struct {
	Control chan *Client // Control is used to pass client control information within Gordian.
	InBox   chan Message // InBox passes incoming messages from clients to Gordian.
	OutBox  chan Message // OutBox passes outgoing messages from Gordian to clients.
	manage  chan *Client
	clients map[ClientId]*Client
	bufSize int
}

// New constructs an initialized Gordian instance.
func New(bufSize int) *Gordian {
	g := &Gordian{
		Control: make(chan *Client),
		InBox:   make(chan Message, bufSize),
		OutBox:  make(chan Message, bufSize),
		manage:  make(chan *Client),
		clients: make(map[ClientId]*Client),
		bufSize: bufSize,
	}
	return g
}

// Run starts Gordian's event loop.
func (g *Gordian) Run() {
	go func() {
		for {
			select {
			case msg := <-g.OutBox:
				if client, ok := g.clients[msg.To]; ok {
					select {
					case client.outBox <- msg:
					default:
					}
				}
			case client := <-g.manage:
				switch client.Ctrl {
				case Establish:
					g.clients[client.Id] = client
				case Close:
					close(client.outBox)
					delete(g.clients, client.Id)
				}
			}
		}
	}()
}

// ServeHTTP handles a websocket connection
func (g *Gordian) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	g.Control <- &Client{Ctrl: Connect, Conn: conn, Request: r}
	client := <-g.Control
	if client.Id == nil || client.Ctrl != Register {
		client.Ctrl = Abort
		g.Control <- client
		return
	}
	client.outBox = make(chan Message, g.bufSize)
	client.Ctrl = Establish
	g.manage <- client
	g.Control <- client

	go g.writeToWS(client)
	g.readFromWS(client)

	client.Ctrl = Close
	g.Control <- client
	g.manage <- client
}

// readFromWS reads a client websocket message and passes it into the system.
func (g *Gordian) readFromWS(client *Client) {
	for {
		jsonMsg := map[string]json.RawMessage{}
		err := websocket.ReadJSON(client.Conn, &jsonMsg)
		if err != nil {
			if err != io.EOF {
				fmt.Println(err)
			}
			return
		}
		typeStr := ""
		err = json.Unmarshal(jsonMsg["type"], &typeStr)
		if err != nil {
			break
		}
		msg := Message{
			From: client.Id,
			Type: typeStr,
			Data: jsonMsg["data"],
		}
		g.InBox <- msg
	}
}

// writeToWS sends a message to a client's websocket.
func (g *Gordian) writeToWS(client *Client) {
	for {
		msg, ok := <-client.outBox
		if !ok {
			return
		}
		jsonMsg := map[string]interface{}{
			"type": msg.Type,
			"data": msg.Data,
		}
		if err := websocket.WriteJSON(client.Conn, jsonMsg); err != nil {
			fmt.Println(err)
		}
	}
}
