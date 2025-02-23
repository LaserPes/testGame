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
		var id int
		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling id: %v", err)
			return
		}
		if err := json.Unmarshal(data, &id); err != nil {
			log.Printf("Error unmarshaling player id: %v", err)
			return
		}
		log.Println("New player ID:", id)
		playerID = id

	case "states_update":
		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling players states: %v", err)
			return
		}
		if err := json.Unmarshal(data, &statePlayers); err != nil {
			log.Printf("Error unmarshaling players states: %v", err)
			return
		}
		mu.Lock()
		for id, state := range statePlayers {
			// Skip our own state
			if id == playerID {
				continue
			}

			// Access map fields correctly
			pos := pixel.V(0, 0)
			if posMap, ok := state["pos"].(map[string]interface{}); ok {
				if x, ok := posMap["X"].(float64); ok {
					if y, ok := posMap["Y"].(float64); ok {
						pos = pixel.V(x, y)
					}
				}
			}

			dir := pixel.V(1, 0)
			if dirMap, ok := state["direction"].(map[string]interface{}); ok {
				if x, ok := dirMap["X"].(float64); ok {
					if y, ok := dirMap["Y"].(float64); ok {
						dir = pixel.V(x, y)
					}
				}
			}

			nickname := ""
			if n, ok := state["nickname"].(string); ok {
				nickname = n
			}

			var heroClass PlayerClass
			if h, ok := state["heroClass"].(string); ok {
				switch h {
				case "warrior":
					heroClass = WarriorClass
				case "mage":
					heroClass = MageClass
				}
			}
			var isAttacking bool
			if attack, ok := state["isAttacking"].(bool); ok {
				isAttacking = attack
			}

			// Create or update other player
			other, exists := otherPlayers[id]
			if !exists {
				newPlayer := &Player{
					ID:           id,
					imd:          imdraw.New(nil),
					speed:        0.3,
					radius:       15,
					direction:    dir,
					projectiles:  make([]*Projectile, 0),
					meleeEffects: make([]*MeleeEffect, 0),
				}
				other = &OtherPlayer{
					Player:   newPlayer,
					LastSeen: time.Now(),
				}
				otherPlayers[id] = other
			}

			other.Player.pos = pos
			other.Player.direction = dir
			other.Player.nickname = nickname
			other.Player.heroClass = heroClass
			// log.Println("hero class: ", heroClass)
			// other.lastSeen = time.Now()
			// other.isAttacking = isAttacking
			otherPlayers[id] = other

			if isAttacking {
				other.Player.Attack()

				// log.Println("attacking player", player)
			}
		}
		mu.Unlock()

	// case "player_attack":
	// 	var attackingPlayer PlayerState
	// 	data, err := json.Marshal(msg.Content)
	// 	if err != nil {
	// 		log.Printf("(Player attack) Error marshaling state: %v", err)
	// 		return
	// 	}
	// 	if err := json.Unmarshal(data, &attackingPlayer); err != nil {
	// 		log.Printf("(Player attack) Error unmarshaling player state: %v", err)
	// 		return
	// 	}

	// 	// Get ID and check if it's our own player BEFORE locking
	// 	var attackPlayerID float64 // Changed from int to float64 for JSON numbers
	// 	if id, ok := attackingPlayer["id"].(float64); ok {
	// 		attackPlayerID = id
	// 	}

	// 	// Skip processing if it's our own attack
	// 	if int(attackPlayerID) == playerID {
	// 		return
	// 	}
	// 	mu.Lock()
	// 	// Access map fields correctly
	// 	pos := pixel.V(0, 0)
	// 	if posMap, ok := attackingPlayer["pos"].(map[string]interface{}); ok {
	// 		if x, ok := posMap["X"].(float64); ok {
	// 			if y, ok := posMap["Y"].(float64); ok {
	// 				pos = pixel.V(x, y)
	// 			}
	// 		}
	// 	}
	// 	dir := pixel.V(1, 0)
	// 	if dirMap, ok := attackingPlayer["direction"].(map[string]interface{}); ok {
	// 		if x, ok := dirMap["X"].(float64); ok {
	// 			if y, ok := dirMap["Y"].(float64); ok {
	// 				dir = pixel.V(x, y)
	// 			}
	// 		}
	// 	}

	// 	nickname := ""
	// 	if n, ok := attackingPlayer["nickname"].(string); ok {
	// 		nickname = n
	// 	}

	// 	var heroClass PlayerClass
	// 	if h, ok := attackingPlayer["heroClass"].(PlayerClass); ok {
	// 		heroClass = PlayerClass(h)
	// 	}

	// 	// Create or update other player
	// 	other, exists := otherPlayers[int(attackPlayerID)]
	// 	if !exists {
	// 		other.Player = &Player{
	// 			ID:           int(attackPlayerID),
	// 			imd:          imdraw.New(nil),
	// 			speed:        0.3,
	// 			radius:       15,
	// 			projectiles:  make([]*Projectile, 0),
	// 			meleeEffects: make([]*MeleeEffect, 0),
	// 		}
	// 		otherPlayers[int(attackPlayerID)] = other
	// 	}

	// 	other.Player.pos = pos
	// 	other.Player.direction = dir
	// 	other.Player.nickname = nickname
	// 	other.Player.heroClass = heroClass
	// 	// other.lastSeen = time.Now()
	// 	other.Player.Attack()
	// 	log.Println("attack player ", player)
	// 	mu.Unlock()

	case "player_left":
		// Handle player leaving messages
		delete(otherPlayers, msg.ClientID)
	}

}

func DrawOtherPlayers(win *pixelgl.Window) {
	currentTime := time.Now()
	var stalePlayers []int

	mu.Lock()
	defer mu.Unlock()

	// First pass: identify stale players and draw active ones
	for id, other := range otherPlayers {
		if other.Player.ID == playerID {
			continue
		}

		// Check if player state is too old (more than 1 second)
		if currentTime.Sub(other.LastSeen) > time.Second {
			stalePlayers = append(stalePlayers, id)
			continue
		}

		// Initialize IMDraw if needed
		if other.Player.imd == nil {
			other.Player.imd = imdraw.New(nil)
		}

		other.Player.Draw(win)
		other.LastSeen = time.Now()

	}

	// Second pass: remove stale players
	for id := range stalePlayers {
		delete(otherPlayers, id)
		otherPlayers[id].Player.imd.Clear()
		log.Printf("Removed stale player: %d", id)
	}
}
