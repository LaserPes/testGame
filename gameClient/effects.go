package main

import (
	"github.com/gopxl/pixel"
	"github.com/gopxl/pixel/imdraw"
)

// Add this new struct at the top with other type declarations
type MeleeEffect struct {
	pos       pixel.Vec
	radius    float64
	imd       *imdraw.IMDraw
	lifetime  float64
	maxLife   float64
	direction pixel.Vec
}

func (m *MeleeEffect) Update() bool {
	m.lifetime += dt
	// m.pos = pos
	m.radius += dt                // Grow the effect over time
	return m.lifetime < m.maxLife // Return false when effect expires
}

func (m *MeleeEffect) Draw(win pixel.Target) {
	m.imd.Clear()
	if !m.Update() {
		delete(meleeAttacks, m)
	} else {
		alpha := 1.0 - (m.lifetime / m.maxLife) // Fade out effect
		m.imd.Color = pixel.RGBA{R: 1, G: 0.2, B: 0.2, A: alpha}
		m.imd.Push(m.pos)
		m.imd.Circle(m.radius, 0)
		m.imd.Draw(win)
	}
}

// Add this function near other constructor functions
func NewMeleeEffect(circle CircleState) *MeleeEffect {
	var pos pixel.Vec
	pos.X = circle.X
	pos.Y = circle.Y
	return &MeleeEffect{
		pos:      pos,
		radius:   circle.Radius,
		imd:      imdraw.New(nil),
		lifetime: 0,
		maxLife:  0.5, // half second duration
	}
}

// Add this to your existing Projectile struct
type Projectile struct {
	pos pixel.Vec
	imd *imdraw.IMDraw
}

func (p *Projectile) Draw(win pixel.Target) {
	p.imd.Clear()
	p.imd.Push(p.pos)
	p.imd.Circle(5, 0)

	p.imd.Color = pixel.RGB(0, 0, 1)

	p.imd.Draw(win)
}

type Explosion struct {
	pos       pixel.Vec
	radius    float64
	maxRadius float64
	lifetime  float64
	maxLife   float64
	imd       *imdraw.IMDraw
}

func NewExplosion(circle CircleState) *Explosion {
	var pos pixel.Vec
	pos.X = circle.X
	pos.Y = circle.Y
	return &Explosion{
		pos:       pos,
		radius:    5,
		maxRadius: circle.Radius,
		lifetime:  0,
		maxLife:   0.2, // 0.3 seconds duration
		imd:       imdraw.New(nil),
	}
}

func (e *Explosion) Update() bool {
	e.lifetime += dt
	if e.lifetime > e.maxLife {
		return false
	}

	// Expand radius until maxRadius
	progress := e.lifetime / e.maxLife
	e.radius = e.maxRadius * progress
	return true
}

func (e *Explosion) Draw(win pixel.Target) {
	e.imd.Clear()
	if !e.Update() {
		delete(explosions, e)
	} else {
		// Fade out as explosion expands
		alpha := 1.0 - (e.lifetime / e.maxLife)
		e.imd.Color = pixel.RGBA{R: 0.2, G: 0.2, B: 1, A: alpha}

		e.imd.Push(e.pos)
		e.imd.Circle(e.radius, 0)
		e.imd.Draw(win)
	}

}
