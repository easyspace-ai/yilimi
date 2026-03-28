package ws

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Event struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Path   string `json:"path"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan Event
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan Event, 128),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				c.Close()
			}
		case evt := <-h.broadcast:
			payload, _ := json.Marshal(evt)
			for c := range h.clients {
				if err := c.WriteMessage(websocket.TextMessage, payload); err != nil {
					log.Printf("ws write error: %v", err)
					c.Close()
					delete(h.clients, c)
				}
			}
		}
	}
}

func (h *Hub) Broadcast(evt Event) {
	select {
	case h.broadcast <- evt:
	default:
		log.Printf("ws broadcast dropped: %+v", evt)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func ServeWS(h *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	h.register <- conn
	go func() {
		defer func() { h.unregister <- conn }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}
