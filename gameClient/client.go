package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gopxl/pixel"
	"github.com/gopxl/pixel/imdraw"
	"github.com/gopxl/pixel/pixelgl"
	"github.com/gopxl/pixel/text"
	"github.com/gorilla/websocket"
	"golang.org/x/exp/rand"
)

var playerID int

type Button struct {
	rect  pixel.Rect
	text  *text.Text
	color pixel.RGBA
}

func NewButton(pos pixel.Vec, label string, atlas *text.Atlas, red, green, blue float64) *Button {
	txt := text.New(pos, atlas)
	fmt.Fprintln(txt, label)
	bounds := txt.Bounds()
	return &Button{
		rect:  pixel.R(pos.X, pos.Y, pos.X+bounds.W(), pos.Y+bounds.H()),
		text:  txt,
		color: pixel.RGB(red, green, blue),
	}
}

func (b *Button) Draw(win *pixelgl.Window) {
	imd := imdraw.New(nil)
	imd.Color = b.color
	imd.Push(b.rect.Min, b.rect.Max)
	imd.Rectangle(1) // Draw button outline
	imd.Draw(win)
	b.text.Draw(win, pixel.IM)
}

func (b *Button) IsClicked(win *pixelgl.Window) bool {
	if win.JustPressed(pixelgl.MouseButtonLeft) {
		mousePos := win.MousePosition()
		return b.rect.Contains(mousePos)
	}
	return false
}

func main() {
	pixelgl.Run(run)

}

func run() {
	// Connect to WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()
	// Define your desired aspect ratio
	aspectRatio := float64(4) / float64(3) // 4:3 aspect ratio

	// Get the primary monitor's size
	primaryMonitor := pixelgl.PrimaryMonitor()
	monitorWidth, monitorHeight := primaryMonitor.Size()

	// Calculate the largest window size that fits on the screen while maintaining the aspect ratio
	var winWidth, winHeight float64
	if float64(monitorWidth)/float64(monitorHeight) > aspectRatio {
		winHeight = monitorHeight * 0.8 // Use 80% of the screen height
		winWidth = winHeight * aspectRatio
	} else {
		winWidth = monitorWidth * 0.8 // Use 80% of the screen width
		winHeight = winWidth / aspectRatio
	}

	cfg := pixelgl.WindowConfig{
		Title:     "Victor's Game",
		Bounds:    pixel.R(0, 0, winWidth, winHeight),
		VSync:     true,
		Resizable: false,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}
	// Center the window on the screen
	win.SetPos(pixel.V(
		(monitorWidth-winWidth)/2,
		(monitorHeight-winHeight)/2,
	))
	// Create channels for WebSocket communication
	done := make(chan struct{})
	receive := make(chan Message)
	// Create quit channel for clean shutdown
	quit := make(chan struct{})
	defer close(quit)

	// Start state processor
	// go ProcessStateUpdates(quit)
	// Start goroutine to receive messages
	go func() {
		defer close(done)
		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("read:", err)
				return
			}
			receive <- msg
		}
	}()

	nickname, heroClass := createPlayerForm(win)
	if nickname == "" || heroClass == "" {
		return // Exit if the form was closed without completing
	}
	msg := Message{
		Type: "new_player",
	}

	err = conn.WriteJSON(msg)
	if err != nil {
		log.Println("write:", err)
	}

	// Handle incoming messages
	select {
	case msg := <-receive:
		HandleMessage(msg, nil)
	default:
		// Continue if no message
	}

	var newPlayerClass PlayerClass
	switch heroClass {
	case "warrior":
		newPlayerClass = WarriorClass
	case "mage":
		newPlayerClass = MageClass
	}
	rand.Seed(uint64(time.Now().UnixNano()))
	randomX := 20 + rand.Float64()*(win.Bounds().Max.X-40)
	randomY := 20 + rand.Float64()*(win.Bounds().Max.Y-40)
	player := NewPlayer(pixel.V(randomX, randomY), win.Bounds(), nickname, newPlayerClass)

	player.ID = playerID
	// Game state variables
	lastTime := time.Now()
	fps := 30.0
	dt := 1.0 / fps

	stateTicker := time.NewTicker(time.Second / 30)
	defer stateTicker.Stop()

	// Main game loop
	for !win.Closed() {

		// Calculate delta time
		currentTime := time.Now()
		dt = currentTime.Sub(lastTime).Seconds()
		lastTime = currentTime

		win.Clear(pixel.RGB(0, 0.5, 0.2))

		// Handle player movement
		if win.Pressed(pixelgl.KeyW) || win.Pressed(pixelgl.KeyUp) {
			player.MoveUp()
		}
		if win.Pressed(pixelgl.KeyS) || win.Pressed(pixelgl.KeyDown) {
			player.MoveDown()
		}
		if win.Pressed(pixelgl.KeyA) || win.Pressed(pixelgl.KeyLeft) {
			player.MoveLeft()
		}
		if win.Pressed(pixelgl.KeyD) || win.Pressed(pixelgl.KeyRight) {
			player.MoveRight()
		}

		// Update player direction based on mouse position
		mousePos := win.MousePosition()
		direction := mousePos.Sub(player.pos).Unit()
		player.direction = direction

		// Handle incoming messages
		// Send player state and draw other players at fixed intervals
		select {
		case <-stateTicker.C:

			msg := Message{
				ClientID: playerID,
				Type:     "states_update",
				Content: PlayerState{
					"id":        player.ID,
					"pos":       player.pos,
					"direction": player.direction,
					"nickname":  player.nickname,
					"heroClass": player.heroClass,
				},
			}
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("write:", err)
				return
			} else {
				log.Println("write:", msg)
			}
		default:
		}

		// Process all pending messages
		for {
			select {
			case msg := <-receive:
				HandleMessage(msg, &player)
			default:
				goto processedMessages
			}
		}
	processedMessages:

		// Handle attacks
		if win.JustPressed(pixelgl.MouseButtonLeft) {
			player.Attack()
		}

		// Update and draw melee effects
		remainingEffects := make([]*MeleeEffect, 0)
		for _, effect := range player.meleeEffects {
			if effect.Update(dt, player.pos) {
				effect.Draw(win)
				remainingEffects = append(remainingEffects, effect)
			}
		}
		player.meleeEffects = remainingEffects

		// Update and draw projectiles
		remainingProjectiles := make([]*Projectile, 0)
		for _, proj := range player.projectiles {
			if proj.Update() {
				proj.Draw(win)
				remainingProjectiles = append(remainingProjectiles, proj)
			}
		}
		player.projectiles = remainingProjectiles
		// DrawOtherPlayers(win)
		DrawOtherPlayers(win)
		// Draw player
		player.Draw(win)

		win.Update()

	}

}
