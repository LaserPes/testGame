package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof" // Import for side effects only
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Id   int
}

type Message struct {
	ClientID int         `json:"client_id"`
	Type     string      `json:"type"`
	Content  interface{} `json:"content"`
}

// Add these constants at the top of the file
const (
	broadcastQueueSize = 512
	readBufferSize     = 1024
	writeBufferSize    = 1024
	maxMessageSize     = 4096
)

var (
	mu           sync.Mutex
	clients      = make(map[*Client]bool)
	latestStates = make(map[int]interface{}) // Хранение последнего состояния каждого игрока
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// Add a buffered channel for broadcasts
var broadcast = make(chan Message, broadcastQueueSize)

func main() {

	errChan := make(chan error, 1)

	// Add pprof endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	go handleMessages(clients, broadcast, errChan)

	go broadcastLatestStates()
	ID := 1
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		// Set connection properties
		conn.SetReadLimit(maxMessageSize)

		mu.Lock()
		client := &Client{
			Conn: conn,
			Id:   ID,
		}
		ID++
		clients[client] = true
		go handleClientStates(client, clients, broadcast, errChan)
		// go handleClientAttacks(client, broadcast, errChan)
		log.Println("New client connected:", client.Id)
		mu.Unlock()

		// Send welcome message
		createMsg := Message{
			Type:    "new_player",
			Content: client.Id,
		}
		if err := client.Conn.WriteJSON(createMsg); err != nil {
			log.Printf("Error sending create message: %v", err)
			return
		}

	})

	fmt.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handleClientStates(client *Client, clients map[*Client]bool, broadcast chan Message, errChan chan error) {
	defer func() {
		mu.Lock()
		client.Conn.Close()
		delete(clients, client)
		delete(latestStates, client.Id)
		mu.Unlock()
	}()

	ticker := time.NewTicker(time.Second / 30) // Регулируй частоту отправки
	defer ticker.Stop()

	for range ticker.C {
		var msg Message

		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			errChan <- err
			break
		}

		if msg.Type == "states_update" {
			mu.Lock()
			latestStates[client.Id] = msg.Content
			// log.Println("Received player state message:", msg)
			mu.Unlock()
		}

		// go func() {
		// 	for {
		// 		var msg Message

		// 		err := client.Conn.ReadJSON(&msg)
		// 		if err != nil {
		// 			errChan <- err
		// 			break
		// 		}
		// 		if msg.Type == "player_attack" {
		// 			broadcast <- msg
		// 		}

		// 	}
		// }()

	}

	// for {
	// 	var msg Message
	// 	err := client.Conn.ReadJSON(&msg)
	// 	if err != nil {
	// 		errChan <- err
	// 		break
	// 	}

	// }
}

// func handleClientAttacks(client *Client, broadcast chan Message, errChan chan error) {
// 	defer func() {
// 		mu.Lock()
// 		broadcast <- Message{
// 			ClientID: client.Id,
// 			Type:     "player_left",
// 		}
// 		client.Conn.Close()
// 		delete(clients, client)
// 		delete(latestStates, client.Id)
// 		log.Println("Client disconnected:", client.Id)
// 		mu.Unlock()
// 	}()

// }
func handleMessages(clients map[*Client]bool, broadcast chan Message, errChan chan error) {
	for {
		select {
		case message := <-broadcast:

			// // Если канал почти заполнен, читаем лишние сообщения
			// for len(broadcast) > broadcastQueueSize/2 {
			// 	<-broadcast // Удаляем старые сообщения
			// }

			mu.Lock()
			for client := range clients {
				if err := client.Conn.WriteJSON(message); err != nil {
					log.Printf("Error broadcasting to client %d: %v", client.Id, err)
					client.Conn.Close()
					delete(clients, client)
				}
			}
			mu.Unlock()

		case err := <-errChan:
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			}
		}
	}
}

func broadcastLatestStates() {
	ticker := time.NewTicker(time.Second / 30) // Частота отправки
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		if len(latestStates) > 0 {
			for client := range clients {
				// log.Println("Broadcasting to client", latestStates)
				msg := Message{
					Type:    "states_update",
					Content: latestStates,
				}
				if err := client.Conn.WriteJSON(msg); err != nil {
					log.Printf("Error broadcasting to client %d: %v", client.Id, err)
					client.Conn.Close()
					delete(clients, client)
				}

			}

		}
		mu.Unlock()
	}
}
