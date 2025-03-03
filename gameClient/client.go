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
)

var playerID int
var playerExists bool
var playerClass int
var nickname string
var playerHP int
var startX, startY float64

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

const (
	fps = 30.0
)

var dt = 1.0 / fps

func main() {
	pixelgl.Run(run)

}
func connectToServer() (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		// Set proper headers and protocol versions
		Subprotocols: []string{"game-protocol"},
	}

	conn, _, err := dialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		return nil, fmt.Errorf("dial error: %v", err)
	}

	// Set connection parameters
	conn.SetReadLimit(32768)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	return conn, nil
}

func run() {
	// Connect to WebSocket server
	conn, err := connectToServer()
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()
	// Define your desired aspect ratio

	// Get the primary monitor's size
	primaryMonitor := pixelgl.PrimaryMonitor()
	monitorWidth, monitorHeight := primaryMonitor.Size()

	// Calculate the largest window size that fits on the screen while maintaining the aspect ratio
	var winWidth, winHeight = 1152.0, 864.0
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
	// done := make(chan struct{})
	receive := make(chan Message, 100) // Add buffer to prevent blocking

	quit := make(chan struct{})
	defer close(quit)

	// Set connection properties
	conn.SetReadLimit(32768)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	go func() {
		for {
			var msg Message
			if err := conn.ReadJSON(&msg); err != nil {
				log.Println("read error:", err)
				return
			}
			receive <- msg

		}
	}()
	nickname, heroClass := createPlayerForm(win)
	if nickname == "" || heroClass == 0 {
		return // Exit if the form was closed without completing
	}
	var playerData struct {
		HeroClass int    `json:"heroClass"`
		Nickname  string `json:"nickname"`
	}
	playerData.HeroClass = heroClass
	playerData.Nickname = nickname
	msg := Message{
		Type:    "new_player",
		Content: playerData,
	}

	err = conn.WriteJSON(msg)
	if err != nil {
		log.Println("new player write:", err)
	}
	n := 4000
	for !playerExists {
		// Handle incoming messages
		select {
		case msg := <-receive:
			log.Println("new player message:", msg)
			HandleMessage(msg, nil)
		default:
			// Continue if no message
		}
		time.Sleep(time.Second / 600)
		n--
		if n == 0 {
			log.Println("Timed out waiting for player")
			break
		}
	}
	if !playerExists {
		return
	}
	//  waitPlayer()
	player := NewPlayer(pixel.V(startX, startY), win.Bounds(), nickname, playerClass)
	player.ID = playerID
	// Game state variables
	lastTime := time.Now()

	stateTicker := time.NewTicker(time.Second / 20)
	defer stateTicker.Stop()

	// Main game loop
	for !win.Closed() {

		// Calculate delta time
		currentTime := time.Now()
		dt = currentTime.Sub(lastTime).Seconds()
		lastTime = currentTime

		win.Clear(pixel.RGB(0, 0.5, 0.4))
		movingX, movingY := 0, 0

		// Handle player movement
		if win.Pressed(pixelgl.KeyW) || win.Pressed(pixelgl.KeyUp) {
			// player.MoveUp()
			movingY++
		}
		if win.Pressed(pixelgl.KeyS) || win.Pressed(pixelgl.KeyDown) {
			// player.MoveDown()
			movingY--
		}
		if win.Pressed(pixelgl.KeyA) || win.Pressed(pixelgl.KeyLeft) {
			// player.MoveLeft()
			movingX--
		}
		if win.Pressed(pixelgl.KeyD) || win.Pressed(pixelgl.KeyRight) {
			// player.MoveRight()
			movingX++
		}

		// Update player direction based on mouse position
		mousePos := win.MousePosition()
		direction := mousePos
		player.direction = direction
		// var myPlayer = &Player{
		// 	ID:        playerID,
		// 	health:    playerClass.Health,
		// 	heroClass: playerClass,
		// 	direction: direction,
		// 	radius:    15,
		// 	imd:       imdraw.New(nil),
		// }
		// myPlayer.pos.X = startX
		// myPlayer.pos.Y = startY
		// var playerTrack = &OtherPlayer{
		// 	Player:      myPlayer,
		// 	LastSeen:    time.Now(),
		// 	IsAttacking: false,
		// }

		// mu.Lock()
		// otherPlayers[playerID] = playerTrack
		// mu.Unlock()
		// myPlayer.Draw(win)
		// Handle incoming messages
		// Send player state and draw other players at fixed intervals
		select {
		case <-stateTicker.C:

			msg := Message{
				ClientID: playerID,
				Type:     "player_moving",
				Content: map[string]interface{}{
					"id":         playerID,
					"directionX": player.direction.X,
					"directionY": player.direction.Y,
					// "heroClass":  heroClass,
					// "nickname": player.nickname,
					"movingX": movingX,
					"movingY": movingY,
				},
			}
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("states write:", err)
				return

			}
		default:
		}

		// Process all pending messages
		for {
			select {
			case msg := <-receive:
				// log.Println("message:", msg)
				HandleMessage(msg, &player)
			default:
				goto processedMessages
			}
		}
	processedMessages:

		// Handle attacks
		if win.JustPressed(pixelgl.MouseButtonLeft) {
			// Check attack cooldown
			// if time.Since(time.Unix(0, int64(player.lastAttack))).Seconds() < float64(player.heroClass.AttackSpeed)/1000.0 {

			// } else {
			// if player.heroClass.AttackType == "physical" {

			// }
			// player.Atta
			// ck(conn)
			// player.lastAttack = float64(time.Now().UnixNano())

			// Send attack message
			msg := Message{
				ClientID: playerID,
				Type:     "player_attack",
				Content: PlayerState{
					"id":         player.ID,
					"directionX": player.direction.X,
					"directionY": player.direction.Y,
					"nickname":   player.nickname,
					// "heroClass":   heroClass,
					// "isAttacking": true,
				},
			}
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("write:", err)
				return
			}

		}

		// // Draw all players every frame
		DrawOtherPlayers(win)

		win.Update()

	}
}

func waitPlayer() {
	if !playerExists {
		time.Sleep(time.Second / 600)
		waitPlayer()
	}
}
