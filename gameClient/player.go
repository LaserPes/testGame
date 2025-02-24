package main

import (
	"encoding/json"
	"fmt"

	"github.com/gopxl/pixel"
	"github.com/gopxl/pixel/imdraw"
	"github.com/gopxl/pixel/pixelgl"
	"github.com/gopxl/pixel/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// Add JSON tags to the Player struct
type Player struct {
	ID           int            `json:"id"`
	pos          pixel.Vec      `json:"pos"`
	speed        float64        `json:"speed"`
	radius       float64        `json:"radius"`
	imd          *imdraw.IMDraw `json:"-"` // Skip serialization
	bounds       pixel.Rect     `json:"-"` // Skip serialization
	nickname     string         `json:"nickname"`
	heroClass    PlayerClass    `json:"heroClass"`
	direction    pixel.Vec      `json:"direction"`
	projectiles  []*Projectile  `json:"-"` // Skip serialization
	lastAttack   float64        `json:"lastAttack"`
	meleeEffects []*MeleeEffect `json:"-"` // Skip serialization
}

// Add custom JSON marshaling methods
func (p *Player) MarshalJSON() ([]byte, error) {
	type Alias struct {
		ID        int         `json:"id"`
		Pos       pixel.Vec   `json:"pos"`
		Speed     float64     `json:"speed"`
		Radius    float64     `json:"radius"`
		Nickname  string      `json:"nickname"`
		HeroClass PlayerClass `json:"heroClass"`
		Direction pixel.Vec   `json:"direction"`
	}

	return json.Marshal(&Alias{
		ID:        p.ID,
		Pos:       p.pos,
		Speed:     p.speed,
		Radius:    p.radius,
		Nickname:  p.nickname,
		HeroClass: p.heroClass,
		Direction: p.direction,
	})
}

// Add custom JSON unmarshaling method
func (p *Player) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID        int         `json:"id"`
		Pos       pixel.Vec   `json:"pos"`
		Speed     float64     `json:"speed"`
		Radius    float64     `json:"radius"`
		Nickname  string      `json:"nickname"`
		HeroClass PlayerClass `json:"heroClass"`
		Direction pixel.Vec   `json:"direction"`
	}

	aux := &Alias{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	p.ID = aux.ID
	p.pos = aux.Pos
	p.speed = aux.Speed
	p.radius = aux.Radius
	p.nickname = aux.Nickname
	p.heroClass = aux.HeroClass
	p.direction = aux.Direction

	// Initialize non-serializable fields
	p.imd = imdraw.New(nil)
	p.projectiles = make([]*Projectile, 0)
	p.meleeEffects = make([]*MeleeEffect, 0)

	return nil
}

type PlayerClass struct {
	MagicResistance    float64
	PhysicalResistance float64
	Health             int
	Attack             int
	AttackRange        float64
	AttackSpeed        int
	AttackType         string
}

var MageClass = PlayerClass{
	MagicResistance:    0.3,
	PhysicalResistance: 0,
	Health:             100,
	Attack:             30,
	AttackRange:        200,
	AttackSpeed:        200,
	AttackType:         "magic",
}

var WarriorClass = PlayerClass{
	MagicResistance:    0,
	PhysicalResistance: 0.5,
	Health:             150,
	Attack:             20,
	AttackRange:        50,
	AttackSpeed:        500,
	AttackType:         "physical",
}

func NewPlayer(pos pixel.Vec, bounds pixel.Rect, nickname string, heroClass PlayerClass) Player {
	return Player{
		pos:          pos,
		speed:        0.3,
		radius:       15,
		imd:          imdraw.New(nil),
		bounds:       bounds,
		nickname:     nickname,
		heroClass:    heroClass,
		direction:    pixel.V(1, 0), // Default direction pointing right
		projectiles:  make([]*Projectile, 0),
		meleeEffects: make([]*MeleeEffect, 0),
		lastAttack:   0,
	}
}

func (p *Player) Draw(win *pixelgl.Window) {
	// Initialize IMDraw if it's nil
	if p.imd == nil {
		p.imd = imdraw.New(nil)
	}

	// Clear IMDraw at the start
	p.imd.Clear()

	// Set player color based on hero class before drawing
	if p.heroClass == WarriorClass {
		p.imd.Color = pixel.RGB(1, 0.2, 0.2) // Red for warrior
	} else if p.heroClass == MageClass {
		p.imd.Color = pixel.RGB(0.2, 0.2, 1) // Blue for mage
	}
	// Draw the player circle
	p.imd.Push(p.pos)
	p.imd.Circle(p.radius, 0)

	p.imd.Draw(win)

	// Draw direction indicator (stick) with the same color
	stickEnd := p.pos.Add(p.direction.Scaled(p.radius * 1.5))
	p.imd.Clear()
	p.imd.Push(p.pos, stickEnd)
	p.imd.Line(2)
	p.imd.Draw(win)

	// Draw nickname text with better contrast
	atlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	nicknameText := text.New(pixel.V(
		p.pos.X-float64(len(p.nickname)*3), // Center text horizontally
		p.pos.Y+p.radius+5),                // Place text above player
		atlas)
	if p.ID == playerID {
		nicknameText.Color = pixel.RGB(1, 1, 1)
	} else {
		nicknameText.Color = pixel.RGB(0, 0, 0)
	}

	fmt.Fprintln(nicknameText, p.nickname)
	nicknameText.Draw(win, pixel.IM)

	// Draw projectiles for mage class
	if p.heroClass == MageClass {
		remainingProjectiles := []*Projectile{}
		for _, proj := range p.projectiles {
			if proj.Update() {
				proj.Draw(win)
				remainingProjectiles = append(remainingProjectiles, proj)
			}
		}
		p.projectiles = remainingProjectiles
	}

	// Draw melee effects
	remainingEffects := []*MeleeEffect{}
	for _, effect := range p.meleeEffects {
		if effect.Update(1.0/60.0, p.pos) { // Assuming 60 FPS
			effect.Draw(win)
			remainingEffects = append(remainingEffects, effect)
		}
	}
	p.meleeEffects = remainingEffects
}

// Update the movement methods to adjust the direction
func (p *Player) MoveUp() {
	newY := p.pos.Y + p.speed
	if newY+p.radius <= p.bounds.Max.Y {
		p.pos.Y = newY

	}
}

func (p *Player) MoveDown() {
	newY := p.pos.Y - p.speed
	if newY-p.radius >= p.bounds.Min.Y {
		p.pos.Y = newY

	}
}

func (p *Player) MoveLeft() {
	newX := p.pos.X - p.speed
	if newX-p.radius >= p.bounds.Min.X {
		p.pos.X = newX

	}
}

func (p *Player) MoveRight() {
	newX := p.pos.X + p.speed
	if newX+p.radius <= p.bounds.Max.X {
		p.pos.X = newX

	}
}

func (p *Player) Attack() {

	if p.heroClass.AttackType == "magic" {
		// Create new projectile
		projectile := NewProjectile(
			p.pos,
			p.direction,
			float64(p.heroClass.AttackRange),
		)
		p.projectiles = append(p.projectiles, projectile)
	} else if p.heroClass.AttackType == "physical" {
		effect := NewMeleeEffect(p.pos, p.heroClass.AttackRange)
		p.meleeEffects = append(p.meleeEffects, effect)
	}

}
func createPlayerForm(win *pixelgl.Window) (string, string) {
	// Replace basicfont with custom sized font
	face, err := opentype.Parse(goregular.TTF)
	if err != nil {
		panic(err)
	}

	fontSize := 20.0 // Adjust this value to change font size
	fontFace, err := opentype.NewFace(face, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}

	atlas := text.NewAtlas(fontFace, text.ASCII)
	nicknameText := text.New(pixel.V(400, 500), atlas)
	classText := text.New(pixel.V(410, 380), atlas)
	buttonWarrior := NewButton(pixel.V(400, 350), "Warrior", atlas, 1, 0, 0)
	buttonMage := NewButton(pixel.V(500, 350), "Mage", atlas, 0, 0, 1)

	nickname := ""
	heroClass := ""
	selectedField := "nickname"

	for !win.Closed() {
		win.Clear(pixel.RGB(0.2, 0.2, 0.2))

		nicknameText.Clear()
		classText.Clear()

		fmt.Fprintf(nicknameText, "Nickname: %s", nickname)
		fmt.Fprintf(classText, "Choose class")

		nicknameText.Draw(win, pixel.IM)
		classText.Draw(win, pixel.IM)
		buttonWarrior.Draw(win)
		buttonMage.Draw(win)

		if win.JustPressed(pixelgl.KeyTab) {
			if selectedField == "nickname" {
				selectedField = "class"
			} else {
				selectedField = "nickname"
			}
		}

		if buttonWarrior.IsClicked(win) && nickname != "" {
			heroClass = "warrior"
			return nickname, heroClass
		}
		if buttonMage.IsClicked(win) && nickname != "" {
			heroClass = "mage"
			return nickname, heroClass
		}

		if win.JustPressed(pixelgl.KeyBackspace) {
			if selectedField == "nickname" && len(nickname) > 0 {
				nickname = nickname[:len(nickname)-1]
			}
		}

		if selectedField == "nickname" {
			nickname += win.Typed()
		} else {
			heroClass += win.Typed()
		}

		win.Update()
	}

	return "", ""
}

// Add this new struct at the top with other type declarations
type MeleeEffect struct {
	pos       pixel.Vec
	radius    float64
	imd       *imdraw.IMDraw
	lifetime  float64
	maxLife   float64
	direction pixel.Vec
}

func (m *MeleeEffect) Update(dt float64, pos pixel.Vec) bool {
	m.lifetime += dt
	m.pos = pos
	m.radius += dt                // Grow the effect over time
	return m.lifetime < m.maxLife // Return false when effect expires
}

func (m *MeleeEffect) Draw(win pixel.Target) {
	m.imd.Clear()
	alpha := 1.0 - (m.lifetime / m.maxLife) // Fade out effect
	m.imd.Color = pixel.RGBA{R: 1, G: 0.2, B: 0.2, A: alpha}
	m.imd.Push(m.pos)
	m.imd.Circle(m.radius, 0)
	m.imd.Draw(win)
}

// Add this function near other constructor functions
func NewMeleeEffect(pos pixel.Vec, r float64) *MeleeEffect {
	return &MeleeEffect{
		pos:      pos,
		radius:   r,
		imd:      imdraw.New(nil),
		lifetime: 0,
		maxLife:  0.5, // half second duration

	}
}

// Add this to your existing Projectile struct
type Projectile struct {
	pos       pixel.Vec
	speed     float64
	radius    float64
	imd       *imdraw.IMDraw
	direction pixel.Vec
	distance  float64
	maxRange  float64
}

// Add this function to create new projectiles
func NewProjectile(pos pixel.Vec, direction pixel.Vec, maxRange float64) *Projectile {
	return &Projectile{
		pos:       pos,
		speed:     0.8,
		radius:    5,
		imd:       imdraw.New(nil),
		direction: direction,
		distance:  0,
		maxRange:  maxRange,
	}
}

// Add this method to draw and update projectiles
func (p *Projectile) Update() bool {
	// Move projectile
	movement := p.direction.Scaled(p.speed)
	p.pos = p.pos.Add(movement)
	p.distance += movement.Len()

	// Return false if projectile exceeded its range
	return p.distance <= p.maxRange
}

func (p *Projectile) Draw(win pixel.Target) {
	p.imd.Clear()
	p.imd.Push(p.pos)
	p.imd.Circle(p.radius, 0)
	if p.maxRange > 50 { // Mage projectile
		p.imd.Color = pixel.RGB(0, 0, 1)
	} else { // Warrior projectile
		p.imd.Color = pixel.RGB(1, 0, 0)
	}
	p.imd.Draw(win)
}
