package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type Message struct {
	Content string `json:"content"`
}

var (
	clients = make(map[*Client]bool)
	mu      sync.Mutex
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", handleWebSocket)

	fmt.Println("Server starting at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	client := &Client{conn: c}

	mu.Lock()
	clients[client] = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(clients, client)
		mu.Unlock()
	}()

	for {
		var msg Message
		err := wsjson.Read(r.Context(), c, &msg)
		if err != nil {
			log.Printf("read error: %v", err)
			return
		}

		broadcast(msg, client)
	}
}

func broadcast(msg Message, sender *Client) {
	mu.Lock()
	defer mu.Unlock()

	for client := range clients {
		if client != sender {
			client.mu.Lock()
			err := wsjson.Write(context.Background(), client.conn, msg)
			client.mu.Unlock()

			if err != nil {
				log.Printf("write error: %v", err)
				continue
			}
		}
	}
}
