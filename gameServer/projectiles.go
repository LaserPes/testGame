package main

import (
	"sync"
	"time"
)

type ServerProjectile struct {
	ID        int
	OwnerID   int
	Pos       Vec2D
	Direction Vec2D
	Speed     float64
	MaxRange  float64
	Distance  float64
	CreatedAt time.Time
}

type Vec2D struct {
	X float64
	Y float64
}

type ProjectileManager struct {
	projectiles map[int]*ServerProjectile
	nextID      int
	mu          sync.Mutex
}

func NewProjectileManager() *ProjectileManager {
	return &ProjectileManager{
		projectiles: make(map[int]*ServerProjectile),
		nextID:      1,
	}
}

func (pm *ProjectileManager) AddProjectile(ownerID int, pos, dir Vec2D, maxRange float64) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	id := pm.nextID
	pm.nextID++

	proj := &ServerProjectile{
		ID:        id,
		OwnerID:   ownerID,
		Pos:       pos,
		Direction: dir,
		Speed:     0.8,
		MaxRange:  maxRange,
		Distance:  0,
		CreatedAt: time.Now(),
	}

	pm.projectiles[id] = proj
	return id
}

// func (pm *ProjectileManager) Update() {
// 	pm.mu.Lock()
// 	defer pm.mu.Unlock()

// 	for id, proj := range pm.projectiles {
// 		// Update projectile position
// 		movement := Vec2D{
// 			X: proj.Direction.X * proj.Speed,
// 			Y: proj.Direction.Y * proj.Speed,
// 		}
// 		proj.Pos.X += movement.X
// 		proj.Pos.Y += movement.Y

// 		// Update distance traveled
// 		proj.Distance += (movement.X*movement.X + movement.Y*movement.Y) * 0.5
// 		circle := Circle{
// 			X:      proj.Pos.X,
// 			Y:      proj.Pos.Y,
// 			Radius: 0.5,
// 		}
// 		mu.Lock()
// 		for id, player := range latestStates {
// 			if id == proj.OwnerID {
// 				continue
// 			}
// 			playerCircle:=Circle{
// 				X:      player.GetPos().X,
//                 Y:      player.GetPos().Y,
//                 Radius: 0.5,
// 			}
// 			if circle.Intersects(player.GetCircle()) {
// 				other.TakeDamage(1)
// 			}
// 		}
// 		mu.Unlock()
// 		// Remove projectile if it exceeded max range
// 		if proj.Distance >= proj.MaxRange {
// 			delete(pm.projectiles, id)
// 		}
// 	}
// }

func (pm *ProjectileManager) GetProjectilesState() map[int]*ServerProjectile {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	state := make(map[int]*ServerProjectile, len(pm.projectiles))
	for id, proj := range pm.projectiles {
		state[id] = proj
	}
	return state
}
