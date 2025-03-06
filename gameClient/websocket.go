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
type ProjectileState struct {
	PosX float64 `json:"posX"`
	PosY float64 `json:"posY"`
}
type CircleState struct {
	X, Y   float64
	Radius float64
}

var stopPlaying bool
var explosions = make(map[*Explosion]bool)
var meleeAttacks = make(map[*MeleeEffect]bool)

var otherPlayers = make(map[int]*OtherPlayer)
var projectiles = make(map[int]Projectile)
var nextExplosionID, nextMeleeID int

type PlayerState map[string]interface{}

var mu, emu, pmu, mmu sync.Mutex

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

			if hp, ok := content["HP"].(int); ok {
				playerHP = hp
			}

		}
		playerExists = true
		log.Println("New player with ID:", playerID, " class:", playerClass, "position:", startX, startY)
	case "states_update":
		var statePlayers = make(map[int]PlayerState)
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
			if id == playerID {
				playerHP = health
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
			log.Printf("Player %d left", msg.ClientID)
		}
		mu.Unlock()
	case "projectiles_update":
		// log.Println(msg)
		var projStates map[int]ProjectileState
		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling players states: %v", err)
			return
		}
		// log.Println(string(data))
		if err := json.Unmarshal(data, &projStates); err != nil {
			log.Printf("Error unmarshaling players states: %v", err)
			return
		}

		for id, state := range projStates {
			pmu.Lock()
			proj, exists := projectiles[id]
			if !exists {
				projectiles[id] = Projectile{
					imd: imdraw.New(nil),
					pos: pixel.V(state.PosX, state.PosY),
				}

			} else {
				proj.pos = pixel.V(state.PosX, state.PosY)
				projectiles[id] = proj
			}
			pmu.Unlock()

		}
		pmu.Lock()
		for id := range projectiles {
			_, exist := projStates[id]
			if !exist {
				delete(projectiles, id)
			}
		}
		pmu.Unlock()
		// log.Println("Projectiles: ", projStates)

	case "explosion_state":

		var expState CircleState
		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling blow state: %v", err)
			return
		}

		if err := json.Unmarshal(data, &expState); err != nil {
			log.Printf("Error unmarshaling blow state: %v", err)
			return
		}
		explosion := NewExplosion(expState)
		emu.Lock()
		explosions[explosion] = true
		// log.Println("explosions:", explosion)
		emu.Unlock()
		nextExplosionID++
	case "melee_state":
		var meleeState CircleState
		data, err := json.Marshal(msg.Content)
		if err != nil {
			log.Printf("Error marshaling blow state: %v", err)
			return
		}

		if err := json.Unmarshal(data, &meleeState); err != nil {
			log.Printf("Error unmarshaling blow state: %v", err)
			return
		}
		melee := NewMeleeEffect(meleeState)
		mmu.Lock()
		meleeAttacks[melee] = true
		mmu.Unlock()
		nextMeleeID++
	case "player_died":
		mu.Lock()
		if playerID == msg.ClientID {
			stopPlaying = true
			playerExists = false
		}
		if other, exists := otherPlayers[msg.ClientID]; exists {
			if other.Player != nil && other.Player.imd != nil {
				other.Player.imd.Clear()
			}
			delete(otherPlayers, msg.ClientID)
			log.Printf("Player %d died", msg.ClientID)
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

		if currentTime.Sub(other.LastSeen) > time.Second {
			stalePlayers = append(stalePlayers, id)
			continue
		}
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
			log.Printf("Removed stale player: %d", id)
		}
	}
}

func DrawProjectiles(win *pixelgl.Window) {
	pmu.Lock()
	for _, proj := range projectiles {
		proj.Draw(win)
	}
	pmu.Unlock()
}

func DrawExplosions(win *pixelgl.Window) {
	emu.Lock()
	for exp := range explosions {
		exp.Draw(win)
	}
	emu.Unlock()
}
func DrawMeleeEffects(win *pixelgl.Window) {
	mmu.Lock()
	for melee := range meleeAttacks {
		melee.Draw(win)
	}
	mmu.Unlock()
}
