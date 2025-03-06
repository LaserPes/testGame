package main

import (
	"log"
	"math"
	"sync"
	"time"
)

type ServerProjectile struct {
	ID        int
	OwnerID   int
	Pos       Vec2D
	Direction Vec2D
	Vec       Vec2D
	Speed     float64
	MaxRange  float64
	Distance  float64
	CreatedAt time.Time
}
type ProjectileState struct {
	PosX float64 `json:"posX"`
	PosY float64 `json:"posY"`
}

type Vec2D struct {
	X float64
	Y float64
}

var nextID int
var projectiles = make(map[int]ServerProjectile)
var pmu sync.Mutex

//	func NewProjectileManager() ProjectileManager {
//		return ProjectileManager{
//			projectiles: make(map[int]*ServerProjectile),
//			nextID:      1,
//		}
//	}
func AddProjectile(ownerID int, pos, dir Vec2D, maxRange float64) int {
	// log.Println("adding projectile:", ownerID, pos, dir, maxRange)
	pmu.Lock()
	defer pmu.Unlock()

	id := nextID
	nextID++

	proj := ServerProjectile{
		ID:        id,
		OwnerID:   ownerID,
		Pos:       pos,
		Direction: dir,
		Speed:     10,
		MaxRange:  maxRange,
		Distance:  0,
		CreatedAt: time.Now(),
	}

	projectiles[id] = proj
	// log.Println("projectile manager: ", projectiles)
	return id
}
func AddMelee(ownerID int, pos Vec2D, maxRange float64) {
	circle := Circle{X: pos.X, Y: pos.Y, Radius: maxRange}
	broadcast <- Message{
		Type:    "melee_state",
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

				attack = attack - (attack * classMap[latestStates[player.ID].HeroClass].MagicResistance)
				player.Health -= attack

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

// Функция для вычисления нормализованного вектора по двум точкам
func NormalizedVector(a, b Vec2D) Vec2D {
	dx := b.X - a.X
	dy := b.Y - a.Y
	length := math.Sqrt(dx*dx + dy*dy) // Вычисляем длину вектора

	if length == 0 { // Проверка на деление на 0 (если точки совпадают)
		return Vec2D{
			X: 0,
			Y: 0,
		}
	}

	return Vec2D{
		X: dx / length,
		Y: dy / length,
	}
}
func projUpdate() {
	pmu.Lock()
	defer pmu.Unlock()

	for projID, proj := range projectiles {
		// Update projectile position
		// vector, exist :=
		var vec Vec2D
		if proj.Vec == vec {
			proj.Vec = NormalizedVector(proj.Pos, proj.Direction)

		}
		movement := Vec2D{
			X: proj.Vec.X * proj.Speed,
			Y: proj.Vec.Y * proj.Speed,
		}
		proj.Pos.X += movement.X
		proj.Pos.Y += movement.Y
		// Update distance traveled
		proj.Distance += math.Sqrt(movement.X*movement.X + movement.Y*movement.Y)
		projectiles[projID] = proj

		mu.Lock()
		for playerID, player := range latestStates {
			if player.ID == proj.OwnerID {
				continue
			}
			circle := Circle{
				X:      proj.Pos.X,
				Y:      proj.Pos.Y,
				Radius: 5,
			}
			playerCircle := Circle{
				X:      player.PosX,
				Y:      player.PosY,
				Radius: 15,
			}
			if circle.Intersects(playerCircle) {
				// Get owner's class for damage calculation
				if _, exists := latestStates[proj.OwnerID]; exists {
					// Update player state

					latestStates[playerID] = player // Save updated state

					// log.Printf("Player %d hit by projectile %d from player %d for %d damage", playerID, projID, proj.OwnerID, classMap[owner.HeroClass].Attack)

					// Remove projectile after hit
					delete(projectiles, projID)
					circle.Radius = 30
					SendExplosion(proj.OwnerID, circle) // Send hit effect

					// Break inner loop since projectile is destroyed
					break
				}
			}
		}
		mu.Unlock()
		// Remove projectile if it exceeded max range
		if proj.Distance >= proj.MaxRange {
			blowUp := Circle{
				X:      proj.Pos.X,
				Y:      proj.Pos.Y,
				Radius: 30,
			}
			SendExplosion(proj.OwnerID, blowUp)
			delete(projectiles, projID)

		}

		log.Println("projectile:", proj)
	}
}

// func GetProjectilesStates() map[int]ServerProjectile {
// 	mu.Lock()
// 	defer mu.Unlock()

// 	state := make(map[int]ServerProjectile, len(projectiles))
// 	for id, proj := range projectiles {
// 		state[id] = proj
// 	}
// 	return state
// }
