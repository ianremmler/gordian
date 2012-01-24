package chatter

import (
	"websocket"
	"strings"
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
	c.clients[id] = struct{}{}
	return id
}

func (c *Chatter) Disconnect(id rtw.ClientId) {
	delete(c.clients, id)
}

func (c *Chatter) Message(msg rtw.Message) {
	for id, _ := range c.clients {
		c.Send(id, msg)
	}
}
