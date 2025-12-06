package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/diabeney/balto/pkg/broadcast"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn   *websocket.Conn
	send   chan broadcast.Event
	mu     sync.Mutex
	closed bool
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for event := range c.send {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()

		if err := c.conn.WriteJSON(event); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
		c.conn.Close()
	}
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan broadcast.Event
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan broadcast.Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.close()
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.mu.RLock()
			clientsToRemove := make([]*Client, 0)
			for client := range h.clients {
				select {
				case client.send <- event:
				default:
					close(client.send)
					clientsToRemove = append(clientsToRemove, client)
				}
			}
			h.mu.RUnlock()

			if len(clientsToRemove) > 0 {
				h.mu.Lock()
				for _, client := range clientsToRemove {
					delete(h.clients, client)
				}
				h.mu.Unlock()
			}
		}
	}
}

var globalHub *Hub
var hubOnce sync.Once

func GetHub() *Hub {
	hubOnce.Do(func() {
		globalHub = NewHub()
		go globalHub.run()
	})
	return globalHub
}

func Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan broadcast.Event, 256),
	}

	hub := GetHub()
	hub.register <- client

	go client.writePump()

	readPump(client, hub)
}

func readPump(client *Client, hub *Hub) {
	defer func() {
		hub.unregister <- client
	}()

	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func SetupBroadcastSubscription(broadcaster *broadcast.Broadcaster) {
	if broadcaster == nil {
		return
	}

	hub := GetHub()
	broadcaster.Subscribe(func(event broadcast.Event) {
		select {
		case hub.broadcast <- event:
		default:
		}
	})
}
