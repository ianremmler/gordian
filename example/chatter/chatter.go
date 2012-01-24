package chatter

import (
	"encoding/json"
	"fmt"
	"github.com/ianremmler/gordian"
	"strings"
	"websocket"
)

type Chatter struct {
	clients map[gordian.ClientId]struct{}
	*gordian.Gordian
}

func NewChatter() *Chatter {
	c := &Chatter{
		clients: make(map[gordian.ClientId]struct{}),
	}
	c.Gordian = gordian.NewGordian(c)
	return c
}

func (c *Chatter) Connect(ws *websocket.Conn) gordian.ClientId {
	path := ws.Request().URL.Path
	id := path[strings.LastIndex(path, "/")+1:]
	if id == "" {
		return nil
	}
	c.clients[id] = struct{}{}
	return id
}

func (c *Chatter) Disconnect(id gordian.ClientId) {
	delete(c.clients, id)
}

func (c *Chatter) Message(msg gordian.Message) {
	var msgJson map[string]string
	if err := json.Unmarshal(msg.Message, &msgJson); err != nil {
		fmt.Println(err)
		return
	}
	if in, ok := msgJson["data"]; ok {
		if idStr, ok := msg.Id.(string); ok {
			msgJson["data"] = idStr + ": " + in
			if out, err := json.Marshal(msgJson); err == nil {
				msg.Message = out
			}
		}
	}
	for id, _ := range c.clients {
		c.Send(id, msg)
	}
}
