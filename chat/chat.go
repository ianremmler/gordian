package main

import (
	"gordian/chatter"
	"net/http"
	"websocket"
)

const (
	htmlDir = "/home/ian/devel/go/src/gordian/chatter/html"
)

func main() {
	c := chatter.NewChatter()
	c.Run()

	http.Handle("/chat/", websocket.Handler(c.WSHandler()))
	http.Handle("/", http.FileServer(http.Dir(htmlDir)))
	if err := http.ListenAndServe(":12345", nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
