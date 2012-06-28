package sim

import (
	"code.google.com/p/go.net/websocket"
	"github.com/ianremmler/gordian"

	"strconv"
	"time"
)


type Sim struct {
	clients    map[gordian.ClientId]struct{}
	ticker     <-chan time.Time
	*gordian.Gordian
}

func New() *Sim {
	s := &Sim{
		clients:    make(map[gordian.ClientId]struct{}),
		// ticker:     time.Tick(10 * 1000000),
		ticker:     time.Tick(1 * 1000000),
		Gordian: gordian.New(),
	}
	return s
}

func (s *Sim) Run() {
	go s.sim()
	s.Gordian.Run()
}

func (s *Sim) sim() {
	msg := &gordian.Message{}
	data := map[string]interface{}{
		"type": "message",
		"data": "",
	}

	i := 0
	for {
		select {
		case ws := <-s.Connect:
			id, err := s.connect(ws)
			s.Manage <- &gordian.ClientInfo{id, err == nil}
		case ci := <-s.Manage:
			if !ci.IsAlive {
				s.disconnect(ci.Id)
			}
		case <-s.ticker:
			data["data"] = strconv.Itoa(i)
			msg.Data = data
			for id, _ := range s.clients {
				msg.To = id
				s.send(msg)
			}
			i++
		}
	}
}

func (s *Sim) connect(ws *websocket.Conn) (gordian.ClientId, error) {
	id := len(s.clients) + 1
	s.clients[id] = struct{}{}
	return id, nil
}

func (s *Sim) disconnect(id gordian.ClientId) {
	delete(s.clients, id)
}

func (s *Sim) send(msg *gordian.Message) {
	s.Messages <- msg
}
