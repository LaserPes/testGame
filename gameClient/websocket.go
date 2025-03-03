package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gopxl/pixel"
	"github.com/gopxl/pixel/imdraw"
	"github.com/gopxl/pixel/pixelgl"
)

type Message struct {
	ClientID int         `json:"client_id"`
	Type     string      `json:"type"`
	Content  interface{} `json:"content"`
}

type OtherPlayer struct {
	Player      *Player
	LastSeen    time.Time
	IsAttacking bool
}

var statePlayers = make(map[int]PlayerState)
var otherPlayers = make(map[int]*OtherPlayer)

type PlayerState map[string]interface{}

var mu sync.Mutex

// Add these constants at the top of the file
const (
	stateProcessingInterval = time.Second / 60 // Process 60 states per second
	stateBufferSize         = 64               // Buffer size for state messages
)

// Add a buffered channel for state processing
var stateUpdateChan = make(chan Message, stateBufferSize)
var clientWindow *pixelgl.Window

func HandleMessage(msg Message, player *Player) {
	switch msg.Type {
	case "new_player":
		contentMap := make(map[string]interface{})
		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling id: %v", err)
			return
		}
		if err := json.Unmarshal(data, &contentMap); err != nil {
			log.Printf("Error unmarshaling player id: %v", err)
			return
		}
		if content, ok := msg.Content.(map[string]interface{}); ok {
			if id, ok := content["id"].(float64); ok {
				playerID = int(id)
			}
			if x, ok := content["X"].(float64); ok {
				startX = x
			}
			if y, ok := content["Y"].(float64); ok {
				startY = y
			}
			// if classID, ok := content["heroClass"].(int); ok {
			// 	playerClass = classID
			// }
			if hp, ok := content["HP"].(int); ok {
				playerHP = hp
			}
			// classJSON, err := json.Marshal(content["heroClass"])
			// if err != nil {
			// 	log.Fatal("JSON Marshal error:", err)
			// }

			// Десериализуем player обратно в структуру PlayerClass

			// err = json.Unmarshal(classJSON, &playerClass)
			// if err != nil {
			// 	log.Fatal("JSON Unmarshal error for player:", err)
			// }

		}
		playerExists = true
		log.Println("New player with ID:", playerID, " class:", playerClass, "position:", startX, startY)
	case "states_update":
		// log.Println(msg)
		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling players states: %v", err)
			return
		}
		// log.Println(string(data))
		if err := json.Unmarshal(data, &statePlayers); err != nil {
			log.Printf("Error unmarshaling players states: %v", err)
			return
		}

		mu.Lock()
		for id, state := range statePlayers {
			// log.Println("got state: ", state)
			// Skip our own state
			if id == playerID {
				playerHP = player.health
			}

			// Access map fields correctly
			pos := pixel.V(0, 0)
			if posX, ok := state["posX"].(float64); ok {
				pos.X = posX
			}
			if posY, ok := state["posY"].(float64); ok {
				pos.Y = posY
			}

			dir := pixel.V(1, 0)
			if dirX, ok := state["directionX"].(float64); ok {
				dir.X = dirX
			}

			if dirY, ok := state["directionY"].(float64); ok {
				dir.Y = dirY
			}

			nickname := ""
			if n, ok := state["nickname"].(string); ok {
				nickname = n
			}
			var health int
			if hp, ok := state["health"].(float64); ok {
				health = int(hp)
			}

			var isAttacking bool
			if attack, ok := state["isAttacking"].(bool); ok {
				isAttacking = attack
			}

			var heroClass int
			if hc, ok := state["heroClass"].(float64); ok {
				heroClass = int(hc)
			}
			// Create or update other player
			other, exists := otherPlayers[id]
			if !exists {
				newPlayer := &Player{
					ID:  id,
					imd: imdraw.New(nil),
					// speed:  0.3,
					radius: 15,
					health: health,
				}
				other = &OtherPlayer{
					Player:   newPlayer,
					LastSeen: time.Now(),
				}
			}

			other.Player.pos = pos
			other.Player.direction = dir.Sub(pos).Unit()
			// log.Println(dir.Sub(pos).Unit())
			other.Player.nickname = nickname
			other.Player.heroClass = heroClass

			otherPlayers[id] = other

			if isAttacking {
				other.Player.Attack(nil)

			}

		}
		mu.Unlock()

	case "player_left":
		// Handle player leaving messages
		mu.Lock()
		if other, exists := otherPlayers[msg.ClientID]; exists {
			if other.Player != nil && other.Player.imd != nil {
				other.Player.imd.Clear()
			}
			delete(otherPlayers, msg.ClientID)
			delete(statePlayers, msg.ClientID)
			log.Printf("Player %d left", msg.ClientID)
		}
		mu.Unlock()
	}

}

func DrawOtherPlayers(win *pixelgl.Window) {
	currentTime := time.Now()
	var stalePlayers []int

	mu.Lock()
	defer mu.Unlock()

	// First pass: identify stale players and draw active ones
	for id, other := range otherPlayers {
		// if other.Player.ID == playerID {
		// 	continue
		// }
		// log.Println("Drawing: ", other.Player.ID)
		// Check if player state is too old (more than 1 second)
		if currentTime.Sub(other.LastSeen) > time.Second {
			stalePlayers = append(stalePlayers, id)
			continue
		}

		// Initialize IMDraw if needed
		// if other.Player.imd == nil {
		// 	other.Player.imd = imdraw.New(nil)
		// }
		other.Player.Draw(win)
		other.LastSeen = time.Now()
	}

	// Second pass: remove stale players
	for id := range stalePlayers {
		if other, exists := otherPlayers[id]; exists {
			if other.Player != nil && other.Player.imd != nil {
				other.Player.imd.Clear()
			}
			delete(otherPlayers, id)
			delete(statePlayers, id)
			log.Printf("Removed stale player: %d", id)
		}
	}
}
