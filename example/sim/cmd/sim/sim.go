package main

import (
	"code.google.com/p/go.net/websocket"
	"github.com/ianremmler/gordian/example/sim"

	"net/http"
)

const (
	htmlDir = "/home/ian/devel/go/src/github.com/ianremmler/gordian/example/sim/html"
)

func main() {
	s := sim.New()
	s.Run()

	http.Handle("/sim/", websocket.Handler(s.WSHandler()))
	http.Handle("/", http.FileServer(http.Dir(htmlDir)))
	if err := http.ListenAndServe(":12345", nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
