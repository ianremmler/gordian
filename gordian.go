package gordian

import (
	"code.google.com/p/go.net/websocket"

	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Control types.
const (
	CONNECT = iota
	REGISTER
	ESTABLISH
	ABORT
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
	Type string      // Type is the type of message.
	Data MessageData // Data is the message payload.
}

// Unmarshal decodes json data in an incoming message
func (m *Message) Unmarshal(data interface{}) error {
	jsonData, ok := m.Data.(json.RawMessage)
	if !ok {
		return errors.New("data is not a json.RawMessage")
	}
	return json.Unmarshal(jsonData, data)
}

// Client stores state and control information for a client.
type Client struct {
	Id      ClientId        // Id is a unique identifier.
	Ctrl    int             // Ctrl is the current control type.
	Conn    *websocket.Conn // Conn is the connection info provided by the websocket package.
	message chan Message
}

// Gordian processes and distributes messages and manages clients.
type Gordian struct {
	Control    chan *Client // Control is used to pass client control information within Gordian.
	InMessage  chan Message // InMessage passes incoming messages from clients to Gordian.
	OutMessage chan Message // OutMessage passes outgoing messages from Gordian to clients.
	manage     chan *Client
	clients    map[ClientId]*Client
	bufSize    int
}

// New constructs an initialized Gordian instance.
func New(bufSize int) *Gordian {
	g := &Gordian{
		Control:    make(chan *Client),
		InMessage:  make(chan Message, bufSize),
		OutMessage: make(chan Message, bufSize),
		manage:     make(chan *Client),
		clients:    make(map[ClientId]*Client),
		bufSize:    bufSize,
	}
	return g
}

// Run starts Gordian's event loop.
func (g *Gordian) Run() {
	go func() {
		for {
			select {
			case msg := <-g.OutMessage:
				if client, ok := g.clients[msg.To]; ok {
					select {
					case client.message <- msg:
					default:
					}
				}
			case client := <-g.manage:
				switch client.Ctrl {
				case ESTABLISH:
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
			client.Ctrl = ABORT
			g.Control <- client
			return
		}
		client.message = make(chan Message, g.bufSize)
		client.Ctrl = ESTABLISH
		g.manage <- client
		g.Control <- client

		go g.writeToWS(client)
		g.readFromWS(client)

		client.Ctrl = CLOSE
		g.Control <- client
		g.manage <- client
	}
}

// readFromWS reads a client websocket message and passes it into the system.
func (g *Gordian) readFromWS(client *Client) {
	for {
		jsonMsg := map[string]json.RawMessage{}
		err := websocket.JSON.Receive(client.Conn, &jsonMsg)
		if err != nil {
			if err != io.EOF {
				fmt.Println(err)
			}
			return
		}

		msgType, ok := jsonMsg["type"]
		if !ok {
			break
		}
		typeStr := ""
		err = json.Unmarshal(msgType, &typeStr)
		if err != nil {
			break;
		}
		msgData, ok := jsonMsg["data"]
		if !ok {
			break
		}
		msg := Message{
			From: client.Id,
			Type: typeStr,
			Data: msgData,
		}
		g.InMessage <- msg
	}
}

// writeToWS sends a message to a client's websocket.
func (g *Gordian) writeToWS(client *Client) {
	for {
		msg, ok := <-client.message
		if !ok {
			return
		}
		jsonMsg := map[string]interface{}{
			"type": msg.Type,
			"data": msg.Data,
		};
		if err := websocket.JSON.Send(client.Conn, jsonMsg); err != nil {
			fmt.Println(err)
		}
	}
}
