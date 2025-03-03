package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/pprof" // Import for side effects only
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/exp/rand"
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
	broadcastQueueSize    = 512
	readBufferSize        = 1024
	writeBufferSize       = 1024
	maxMessageSize        = 4096
	MessageTypeProjectile = "projectile"
)

var (
	mu sync.Mutex

	clients      = make(map[*Client]bool)
	latestStates = make(map[int]PlayerState) // Хранение последнего состояния каждого игрока
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

type Circle struct {
	X, Y   float64
	Radius float64
}

func (c1 Circle) Intersects(c2 Circle) bool {
	distance := math.Sqrt(math.Pow(c2.X-c1.X, 2) + math.Pow(c2.Y-c1.Y, 2))
	return distance <= (c1.Radius + c2.Radius)
}

// func main() {
// 	c1 := Circle{X: 0, Y: 0, Radius: 5}
// 	c2 := Circle{X: 7, Y: 0, Radius: 3}

//		if c1.Intersects(c2) {
//			fmt.Println("Окружности пересекаются")
//		} else {
//			fmt.Println("Окружности не пересекаются")
//		}
//	}
//
// Add a buffered channel for broadcasts
var broadcast = make(chan Message, broadcastQueueSize)
var projectileManager = NewProjectileManager()

// Calculate the largest window size that fits on the screen while maintaining the aspect ratio
var winWidth, winHeight = 1152.0, 864.0

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
	go updateProjectiles()
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

		client := &Client{
			Conn: conn,
			Id:   ID,
		}
		ID++
		mu.Lock()
		clients[client] = true
		mu.Unlock()
		// go handleClientAttacks(client, broadcast, errChan)
		log.Println("New client connected:", client.Id)

		var msg Message
		if err := client.Conn.ReadJSON(&msg); err != nil {
			log.Printf("Error sending create message: %v", err)
			return
		}

		var playerData struct {
			HeroClass int    `json:"heroClass"`
			Nickname  string `json:"nickname"`
		}

		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling new player data: %v", err)
			return
		}

		if err := json.Unmarshal(data, &playerData); err != nil {
			log.Printf("Error unmarshaling new player data: %v", err)
			return
		}
		log.Println("New player data: ", playerData)
		rand.Seed(uint64(time.Now().UnixNano()))
		randomX := 20 + rand.Float64()*(winWidth-40)
		randomY := 20 + rand.Float64()*(winHeight-40)
		// Send welcome message
		createMsg := Message{
			Type: "new_player",
			Content: map[string]interface{}{
				"id": client.Id,
				"X":  randomX,
				"Y":  randomY,
				// "heroClass": classMap[playerData.HeroClass].ID,
				"HP": classMap[playerData.HeroClass].Health,
			},
		}
		log.Println(createMsg)
		var newPlayerState = PlayerState{
			ID:        client.Id,
			PosX:      randomX,
			PosY:      randomY,
			HeroClass: playerData.HeroClass,
			Nickname:  playerData.Nickname,
			Health:    classMap[playerData.HeroClass].Health,
		}
		mu.Lock()
		latestStates[client.Id] = newPlayerState
		mu.Unlock()

		jsonData, err := json.Marshal(createMsg)
		if err != nil {
			log.Println("JSON Marshal error:", err)
			return
		}
		log.Println(createMsg)
		// Отправляем данные по WebSocket
		err = conn.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			log.Println("WriteMessage error:", err)
		}
		go handleClientStates(client, clients, broadcast, errChan)
	})

	fmt.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handleClientStates(client *Client, clients map[*Client]bool, broadcast chan Message, errChan chan error) {

	// ticker := time.NewTicker(time.Second / 30) // Регулируй частоту отправки

	for {
		var msg Message

		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			errChan <- err
			break
		}

		switch msg.Type {
		case "player_moving":
			log.Println("player moving: ", msg)
			var movement PlayerMovement
			data, err := json.Marshal(msg.Content)
			if err != nil {
				log.Printf("Error marshaling movement data: %v", err)
				return
			}

			if err := json.Unmarshal(data, &movement); err != nil {
				log.Printf("Error unmarshaling movement data: %v", err)
				return
			}

			// Now you can use the movement data
			mu.Lock()
			if state, exists := latestStates[movement.ID]; exists {
				// Update player position based on movement
				state.PosX += float64(movement.MovingX) * 2 //float64(latestStates[movement.ID].HeroClass.Speed)
				state.PosY += float64(movement.MovingY) * 2 //float64(latestStates[movement.ID].HeroClass.Speed)
				state.DirectionX = movement.DirectionX
				state.DirectionY = movement.DirectionY
				latestStates[movement.ID] = state

			}
			mu.Unlock()

		case "player_attack":
			var attack PlayerAttack
			data, err := json.Marshal(msg.Content)
			if err != nil {
				log.Printf("Error marshaling attack data: %v", err)
				return
			}

			if err := json.Unmarshal(data, &attack); err != nil {
				log.Printf("Error unmarshaling attack data: %v", err)
				return
			}
			if state, exists := latestStates[attack.ID]; exists {
				// Update player position based on movement
				state.DirectionX = attack.DirectionX
				state.DirectionY = attack.DirectionY
				if time.Since(time.Unix(0, int64(state.LastAttack))).Seconds() < float64(classMap[state.HeroClass].AttackSpeed)/1000.0 {
					state.IsAttacking = true
				}

				latestStates[attack.ID] = state

			}
			// case "projectile":
			// 	if content, ok := msg.Content.(map[string]interface{}); ok {
			// 		pos := Vec2D{}
			// 		dir := Vec2D{}
			// 		var maxRange float64

			// 		if posMap, ok := content["pos"].(map[string]interface{}); ok {
			// 			if x, ok := posMap["X"].(float64); ok {
			// 				pos.X = x
			// 			}
			// 			if y, ok := posMap["Y"].(float64); ok {
			// 				pos.Y = y
			// 			}
			// 		}

			// 		if dirMap, ok := content["direction"].(map[string]interface{}); ok {
			// 			if x, ok := dirMap["X"].(float64); ok {
			// 				dir.X = x
			// 			}
			// 			if y, ok := dirMap["Y"].(float64); ok {
			// 				dir.Y = y
			// 			}
			// 		}

			// 		if r, ok := content["maxRange"].(float64); ok {
			// 			maxRange = r
			// 		}

			// 		projectileManager.AddProjectile(client.Id, pos, dir, maxRange)
			// 	}
		}
	}
	defer func() {
		mu.Lock()
		broadcast <- Message{
			ClientID: client.Id,
			Type:     "player_left",
		}
		client.Conn.Close()
		delete(clients, client)
		delete(latestStates, client.Id)
		log.Println("Client disconnected:", client.Id)
		mu.Unlock()
		// ticker.Stop()
	}()

}

func handleMessages(clients map[*Client]bool, broadcast chan Message, errChan chan error) {
	for {
		select {
		case message := <-broadcast:

			mu.Lock()
			for client := range clients {
				if err := client.Conn.WriteJSON(message); err != nil {
					log.Printf("Error broadcasting to client %d: %v", client.Id, err)
					client.Conn.Close()
					delete(clients, client)
					delete(latestStates, client.Id)
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
	ticker := time.NewTicker(time.Second / 20) // Частота отправки
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
				log.Println(latestStates, client)
				data, err := json.Marshal(latestStates)
				if err != nil {
					log.Println("Error marshaling state data: ", err)
				}
				log.Println(string(data))
				if err := client.Conn.WriteJSON(msg); err != nil {
					log.Printf("Error broadcasting to client %d: %v", client.Id, err)
					client.Conn.Close()
					delete(clients, client)
					delete(latestStates, client.Id)
				}

				log.Println("State update: ", msg)

			}

		}
		mu.Unlock()
	}
}
func updateProjectiles() {
	ticker := time.NewTicker(time.Second / 60) // 60 updates per second
	defer ticker.Stop()

	for range ticker.C {
		// projectileManager.Update()

		// Broadcast projectile states to all clients
		state := projectileManager.GetProjectilesState()
		if len(state) > 0 {
			broadcast <- Message{
				Type:    "projectiles_update",
				Content: state,
			}
		}
	}
}
