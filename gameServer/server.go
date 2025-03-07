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
	tickRate              = time.Second / 30
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

	upgrader = websocket.Upgrader{
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

// Add a buffered channel for broadcasts
var broadcast = make(chan Message, broadcastQueueSize)

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
		var newPlayer PlayerData
		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling new player data: %v", err)
			return
		}

		if err := json.Unmarshal(data, &newPlayer); err != nil {
			log.Printf("Error unmarshaling new player data: %v", err)
			return
		}
		log.Println("New player data: ", newPlayer)
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
				"HP": classMap[newPlayer.HeroClass].Health,
			},
		}
		log.Println(createMsg)
		var newPlayerState = PlayerState{
			ID:        client.Id,
			PosX:      randomX,
			PosY:      randomY,
			HeroClass: newPlayer.HeroClass,
			Nickname:  newPlayer.Nickname,
			Health:    classMap[newPlayer.HeroClass].Health,
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
				if 1 >= movement.MovingX && movement.MovingX >= -1 && 1 >= movement.MovingY && movement.MovingY >= -1 {
					if (state.PosX-15) >= 0 && movement.MovingX == -1 {
						state.PosX += float64(movement.MovingX) * (float64(classMap[latestStates[movement.ID].HeroClass].Speed) / 100)
					}
					if (state.PosX+15) <= winWidth && movement.MovingX == 1 {
						state.PosX += float64(movement.MovingX) * (float64(classMap[latestStates[movement.ID].HeroClass].Speed) / 100)
					}
					if (state.PosY-15) >= 0 && movement.MovingY == -1 {
						state.PosY += float64(movement.MovingY) * (float64(classMap[latestStates[movement.ID].HeroClass].Speed) / 100)
					}
					if (state.PosY+15) <= winHeight && movement.MovingY == 1 {
						state.PosY += float64(movement.MovingY) * (float64(classMap[latestStates[movement.ID].HeroClass].Speed) / 100)
					}

					state.DirectionX = movement.DirectionX
					state.DirectionY = movement.DirectionY
					latestStates[movement.ID] = state
					// log.Println("00000", state)
				}
			}
			mu.Unlock()

		case "player_attack":
			log.Println("player attack: ", msg)
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

			// state.IsAttacking = true
			mu.Lock()
			if state, exists := latestStates[attack.ID]; exists {
				if time.Since(state.LastAttack).Seconds() >= float64(classMap[state.HeroClass].AttackSpeed)/1000.0 {
					// Update player position based on movement
					pos := Vec2D{
						X: state.PosX,
						Y: state.PosY,
					}
					dir := Vec2D{
						X: attack.DirectionX,
						Y: attack.DirectionY,
					}
					if classMap[state.HeroClass].AttackType == "magic" {
						log.Println("magic attack:", attack.ID, pos, dir)
						AddProjectile(attack.ID, pos, dir, classMap[state.HeroClass].AttackRange)
					} else if classMap[state.HeroClass].AttackType == "physical" {
						log.Println("melee attack:", attack.ID, pos, dir)
						AddMelee(attack.ID, pos, classMap[state.HeroClass].AttackRange)
					}

					state.DirectionX = attack.DirectionX
					state.DirectionY = attack.DirectionY
					state.LastAttack = time.Now()
					latestStates[attack.ID] = state
				}
			}
			mu.Unlock()
		case "new_player":
			log.Println("new player: ", msg)
			var newPlayer PlayerData
			data, err := json.Marshal(msg.Content)
			if err != nil {
				log.Printf("Error marshaling new player data: %v", err)
				return
			}
			if err := json.Unmarshal(data, &newPlayer); err != nil {
				log.Printf("Error unmarshaling new player data: %v", err)
				return
			}
			mu.Lock()
			for client := range clients {
				if client.Id == msg.ClientID {
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
							"HP": classMap[newPlayer.HeroClass].Health,
						},
					}
					log.Println(createMsg)
					var newPlayerState = PlayerState{
						ID:        client.Id,
						PosX:      randomX,
						PosY:      randomY,
						HeroClass: newPlayer.HeroClass,
						Nickname:  newPlayer.Nickname,
						Health:    classMap[newPlayer.HeroClass].Health,
					}

					latestStates[client.Id] = newPlayerState

					jsonData, err := json.Marshal(createMsg)
					if err != nil {
						log.Println("JSON Marshal error:", err)
						return
					}
					log.Println(createMsg)
					// Отправляем данные по WebSocket
					err = client.Conn.WriteMessage(websocket.TextMessage, jsonData)
					if err != nil {
						log.Println("WriteMessage error:", err)
					}
				}
			}
			mu.Unlock()

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
	ticker := time.NewTicker(tickRate)
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

				// log.Println("State update: ", msg)

			}

		}

		mu.Unlock()
	}
}
func updateProjectiles() {
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()
	var noRepeat bool
	for range ticker.C {
		projUpdate()
		// Broadcast projectile states to all clients
		// states := GetProjectilesStates()
		pmu.Lock()
		//  log.Println("Projectiles len: ", projectiles)

		if len(projectiles) > 0 {
			var projStates = make(map[int]ProjectileState)

			for _, state := range projectiles {

				projStates[state.ID] = ProjectileState{
					PosX: state.Pos.X,
					PosY: state.Pos.Y,
				}

			}
			log.Println("Projectiles updated: ", projStates)

			broadcast <- Message{
				Type:    "projectiles_update",
				Content: projStates,
			}
			noRepeat = false
		} else if !noRepeat {
			noRepeat = true
			broadcast <- Message{
				Type:    "projectiles_update",
				Content: make(map[int]ProjectileState),
			}

		}
		pmu.Unlock()

	}
}

func SendExplosion(ownerID int, circle Circle) {
	broadcast <- Message{
		Type:    "explosion_state",
		Content: circle,
	}
	for playerID, player := range latestStates {

		if player.ID == ownerID {
			continue
		}
		playerCircle := Circle{
			X:      player.PosX,
			Y:      player.PosY,
			Radius: 15,
		}

		if circle.Intersects(playerCircle) {
			// Get owner's class for damage calculation
			if owner, exists := latestStates[ownerID]; exists {
				attackType := classMap[owner.HeroClass].AttackType
				attack := classMap[owner.HeroClass].Attack
				// Update player state
				if attackType == "magic" {
					attack = attack - (attack * classMap[latestStates[player.ID].HeroClass].MagicResistance)
					player.Health -= attack
				}

				if player.Health <= 0 {
					broadcast <- Message{
						ClientID: player.ID,
						Type:     "player_died",
					}

					delete(latestStates, playerID)
					log.Printf("Player %d died", playerID)
					break
				}
				latestStates[playerID] = player // Save updated state

				log.Printf("Player %d hit by %s from player %d for %f damage",
					playerID, attackType, ownerID, attack)
			}
		}
	}

}
