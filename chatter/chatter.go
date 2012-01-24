package chatter

import (
	"websocket"
	"strings"
	"rtw/rtw"
)

type Chatter struct {
	rtw *rtw.RTW
	clients map[rtw.ClientId]struct{}
}

func NewChatter() *Chatter {
	c := &Chatter{
		clients: make(map[rtw.ClientId]struct{}),
	}
	c.rtw = rtw.NewRTW(c)
	return c
}

func (c *Chatter) Run() {
	c.rtw.Run()
}

func (c *Chatter) Handler() func(*websocket.Conn) {
	return c.rtw.WSHandler()
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
		c.rtw.Send(id, msg)
	}
}
