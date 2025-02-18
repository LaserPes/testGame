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

// type PlayerState  {
// 	ID        int         `json:"id"`
// 	Position  pixel.Vec   `json:"position"`
// 	Direction pixel.Vec   `json:"direction"`
// 	Nickname  string      `json:"nickname"`
// 	HeroClass PlayerClass `json:"hero_class"`
// }

var statePlayers = make(map[int]PlayerState)
var otherPlayers = make(map[int]*Player)

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
		// Convert the content map to JSON
	case "states_update":
		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling id: %v", err)
			return
		}
		if err := json.Unmarshal(data, &statePlayers); err != nil {
			log.Printf("Error unmarshaling player id: %v", err)
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
			if h, ok := state["heroClass"].(PlayerClass); ok {
				heroClass = PlayerClass(h)
			}

			// Create or update other player
			other, exists := otherPlayers[id]
			if !exists {
				other = &Player{
					ID:     id,
					imd:    imdraw.New(nil),
					speed:  0.3,
					radius: 15,
				}
				otherPlayers[id] = other
			}

			other.pos = pos
			other.direction = dir
			other.nickname = nickname
			other.heroClass = heroClass
		}
		mu.Unlock()

		log.Println("States update:", statePlayers)

	case "attack":
		// Handle attack messages
	case "player_left":
		// Handle player leaving messages
		delete(otherPlayers, msg.ClientID)
	}

}

func DrawOtherPlayers(win *pixelgl.Window) {
	// currentTime := time.Now()

	// Create a temporary slice to track stale players
	// var stalePlayers []int
	// playersMutex.Lock()
	// Draw and update other players
	mu.Lock()
	for _, other := range otherPlayers {

		// Check if player state is too old (more than 1 second)
		// if currentTime.Sub(other.LastSeen) > time.Second*1 {
		// 	stalePlayers = append(stalePlayers, id)
		// 	continue
		// }
		// Initialize IMDraw if it's nil
		if other.ID == playerID {
			continue
		}
		if other.imd == nil {
			other.imd = imdraw.New(nil)
		}

		other.Draw(win)
		log.Println("Drawing player", other)
	}
	// playersMutex.Unlock()
	// Remove stale players
	// for _, id := range stalePlayers {
	// 	delete(otherPlayers, id)
	// 	log.Printf("Removed stale player %d", id)
	// }
	mu.Unlock()
}

// // Add new function to process state updates
// func ProcessStateUpdates(quit chan struct{}) {
// 	ticker := time.NewTicker(stateProcessingInterval)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			// Process all pending states, but only the most recent one per player
// 			states := make(map[int]Message)

// 			// Drain the channel and keep only the latest state per player
// 		drainLoop:
// 			for {

// 				select {
// 				case msg := <-stateUpdateChan:
// 					if contentMap, ok := msg.Content.(map[string]interface{}); ok {
// 						if id, ok := contentMap["id"].(float64); ok {
// 							states[int(id)] = msg // Override older states
// 						}
// 					}
// 				default:
// 					break drainLoop
// 				}
// 			}

// 			// Process the latest states
// 			for _, msg := range states {
// 				processPlayerState(msg)
// 			}
// 		case <-quit:
// 			return
// 		}
// 	}
// }

// // Add helper function to process individual player state
// func processPlayerState(msg Message) {
// 	contentMap, ok := msg.Content.(map[string]interface{})
// 	if !ok || contentMap == nil {
// 		return
// 	}

// 	idVal, exists := contentMap["id"]
// 	if !exists {
// 		return
// 	}

// 	id, ok := idVal.(float64)
// 	if !ok || int(id) == playerID {
// 		return
// 	}
// 	mu.Lock()
// 	// Get or create other player
// 	other, exists := otherPlayers[int(id)]
// 	if !exists {
// 		other = &OtherPlayer{
// 			Player: Player{
// 				ID:  int(id),
// 				imd: imdraw.New(nil),
// 			},
// 			LastSeen: time.Now(),
// 		}
// 		otherPlayers[int(id)] = other
// 		log.Println("\nGot player", other)
// 	}
// 	mu.Unlock()

// 	// Update player state directly without marshal/unmarshal
// 	if pos, ok := contentMap["pos"].(map[string]interface{}); ok {
// 		if x, ok := pos["X"].(float64); ok {
// 			if y, ok := pos["Y"].(float64); ok {
// 				other.Player.pos = pixel.V(x, y)
// 			}
// 		}
// 	}
// 	if dir, ok := contentMap["direction"].(map[string]interface{}); ok {
// 		if x, ok := dir["X"].(float64); ok {
// 			if y, ok := dir["Y"].(float64); ok {
// 				other.Player.direction = pixel.V(x, y)
// 			}
// 		}
// 	}
// 	if nickname, ok := contentMap["nickname"].(string); ok {
// 		other.Player.nickname = nickname
// 	}
// 	if heroClass, ok := contentMap["heroClass"].(PlayerClass); ok {
// 		other.Player.heroClass = PlayerClass(heroClass)
// 	}
// 	other.LastSeen = time.Now()

// }
