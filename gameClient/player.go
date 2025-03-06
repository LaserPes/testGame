package main

import (
	"encoding/json"
	"fmt"

	"github.com/gopxl/pixel"
	"github.com/gopxl/pixel/imdraw"
	"github.com/gopxl/pixel/pixelgl"
	"github.com/gopxl/pixel/text"
	"github.com/gorilla/websocket"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// Add JSON tags to the Player struct
type Player struct {
	ID         int
	pos        pixel.Vec
	speed      float64
	radius     float64
	imd        *imdraw.IMDraw
	bounds     pixel.Rect
	nickname   string
	heroClass  int
	direction  pixel.Vec
	lastAttack float64
	health     int
}

// Add custom JSON marshaling methods
func (p *Player) MarshalJSON() ([]byte, error) {
	type Alias struct {
		ID        int       `json:"id"`
		Pos       pixel.Vec `json:"pos"`
		Speed     float64   `json:"speed"`
		Radius    float64   `json:"radius"`
		Nickname  string    `json:"nickname"`
		HeroClass int       `json:"heroClass"`
		Direction pixel.Vec `json:"direction"`
	}

	return json.Marshal(&Alias{
		ID:  p.ID,
		Pos: p.pos,
		// Speed:     p.speed,
		Radius:    p.radius,
		Nickname:  p.nickname,
		HeroClass: p.heroClass,
		Direction: p.direction,
	})
}

// Add custom JSON unmarshaling method
func (p *Player) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID        int       `json:"id"`
		Pos       pixel.Vec `json:"pos"`
		Speed     float64   `json:"speed"`
		Radius    float64   `json:"radius"`
		Nickname  string    `json:"nickname"`
		HeroClass int       `json:"heroClass"`
		Direction pixel.Vec `json:"direction"`
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
	p.imd = imdraw.New(nil)

	return nil
}

// Add after other type declarations

func NewPlayer(pos pixel.Vec, bounds pixel.Rect, nickname string, heroClass int) Player {

	return Player{
		pos: pos,
		// speed:     0.3,
		radius:     15,
		imd:        imdraw.New(nil),
		bounds:     bounds,
		nickname:   nickname,
		heroClass:  heroClass,
		direction:  pixel.V(1, 0), // Default direction pointing right
		lastAttack: 0,
		health:     playerHP,
	}
}

func (p *Player) Draw(win *pixelgl.Window) {
	// Initialize IMDraw if it's nil
	if p.imd == nil {
		p.imd = imdraw.New(nil)
	}

	// Clear IMDraw once at the start
	p.imd.Clear()

	// Draw HP circle (inner circle)
	var maxHP int
	if p.heroClass == 1 {
		maxHP = 150
	} else if p.heroClass == 2 {
		maxHP = 100
	}
	red := (float64(maxHP) - float64(p.health)) / float64(maxHP)
	green := float64(p.health) / float64(maxHP)
	p.imd.Color = pixel.RGB(red, green, 0)
	p.imd.Push(p.pos)
	p.imd.Circle(p.radius-5, 0) // Smaller radius for HP indicator

	// Draw outer circle with class color
	if p.heroClass == 1 {
		p.imd.Color = pixel.RGB(1, 0.2, 0.2) // Red for warrior
	} else if p.heroClass == 2 {
		p.imd.Color = pixel.RGB(0.2, 0.2, 1) // Blue for mage
	}
	p.imd.Push(p.pos)
	p.imd.Circle(p.radius, 1) // Use outline for outer circle

	// Draw direction indicator (stick) with the same color as outer circle
	stickEnd := p.pos.Add(p.direction.Scaled(p.radius * 1.5))
	p.imd.Push(p.pos, stickEnd)
	p.imd.Line(2)

	// Draw everything at once
	p.imd.Draw(win)

	// Draw nickname text
	atlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	nicknameText := text.New(pixel.V(
		p.pos.X-float64(len(p.nickname)*3),
		p.pos.Y+p.radius+5),
		atlas)
	if p.ID == playerID {
		nicknameText.Color = pixel.RGB(1, 1, 1)
	} else {
		nicknameText.Color = pixel.RGB(0, 0, 0)
	}
	fmt.Fprintln(nicknameText, p.nickname)
	nicknameText.Draw(win, pixel.IM)
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

func (p *Player) Attack(conn *websocket.Conn) {

	// Send projectile creation message to server
	msg := Message{
		Type: "projectile",
		Content: map[string]interface{}{
			"pos":       p.pos,
			"direction": p.direction,
			// "maxRange":  float64(p.heroClass.AttackRange),
		},
	}
	// You'll need to access the WebSocket connection here
	if p.ID == playerID {
		conn.WriteJSON(msg)
	}

}

func createPlayerForm(win *pixelgl.Window) (string, int) {
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

	heroClass := 0
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
			heroClass = 1
			return nickname, heroClass
		}
		if buttonMage.IsClicked(win) && nickname != "" {
			heroClass = 2
			return nickname, heroClass
		}

		if win.JustPressed(pixelgl.KeyBackspace) {
			if selectedField == "nickname" && len(nickname) > 0 {
				nickname = nickname[:len(nickname)-1]
			}
		}

		if selectedField == "nickname" {
			nickname += win.Typed()
		}

		win.Update()
	}

	return "", 0
}
