package chatter

import (
	"websocket"
	"strings"
	"encoding/json"
	"fmt"
	"rtw/rtw"
)

type Chatter struct {
	clients map[rtw.ClientId]struct{}
	*rtw.RTW
}

func NewChatter() *Chatter {
	c := &Chatter{
		clients: make(map[rtw.ClientId]struct{}),
	}
	c.RTW = rtw.NewRTW(c)
	return c
}

func (c *Chatter) Connect(ws *websocket.Conn) rtw.ClientId {
	path := ws.Request().URL.Path
	id := path[strings.LastIndex(path, "/") + 1:]
	if id == "" {
		return nil
	}
	c.clients[id] = struct{}{}
	return id
}

func (c *Chatter) Disconnect(id rtw.ClientId) {
	delete(c.clients, id)
}

func (c *Chatter) Message(msg rtw.Message) {
	var msgJson map[string]string
	if err := json.Unmarshal(msg.Message, &msgJson); err != nil {
		fmt.Println(err)
		return
	}
	if in, ok := msgJson["data"]; ok {
		if idStr, ok := msg.Id.(string); ok {
			msgJson["data"] = idStr + ": " + in
			if out, err := json.Marshal(msgJson); err != nil {
				return
			} else {
				msg.Message = out
			}
		}
	}
	for id, _ := range c.clients {
		c.Send(id, msg)
	}
}
