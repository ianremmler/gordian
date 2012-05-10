package chatter

import (
	"code.google.com/p/go.net/websocket"
	"github.com/ianremmler/gordian"

	"errors"
	"strings"
)

type Chatter struct {
	clients map[gordian.ClientId]struct{}
	*gordian.Gordian
}

func NewChatter() *Chatter {
	c := &Chatter{
		clients: make(map[gordian.ClientId]struct{}),
	}
	c.Gordian = gordian.New(c)
	return c
}

func (c *Chatter) Connect(ws *websocket.Conn) (gordian.ClientId, error) {
	path := ws.Request().URL.Path
	id := path[strings.LastIndex(path, "/")+1:]
	if id == "" {
		return "", errors.New("Invalid ID")
	}
	c.clients[id] = struct{}{}
	return id, nil
}

func (c *Chatter) Disconnect(id gordian.ClientId) {
	delete(c.clients, id)
}

func (c *Chatter) HandleMessage(msg *gordian.Message) {
	data := msg.Data.(map[string]interface{})
	if in, ok := data["data"].(string); ok {
		data["data"] = msg.Id.(string) + ": " + in
		out := &gordian.Message{msg.Id, data}
		for id, _ := range c.clients {
			c.Send(id, out)
		}
	}
}
