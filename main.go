package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// Ship represents the player's ship
type Ship struct {
	X, Y    float64
	Angle   float64
	Size    float64
	Health  int
}

type EnemyShip struct {
				X, Y    float64
				Angle   float64
				Size    float64
				Health  int
				Speed   float64
}


// Bullet represents a bullet fired by the ship
type Bullet struct {
	X, Y   float64
	Angle  float64
	Damage int
}

// Asteroid represents a falling asteroid with a random shape
type Asteroid struct {
	X, Y        float64
	Vertices    [][2]float64
	NumVertices int
	Size        float64
	Angle       float64
	Speed       float64
	Health      int
}

type GameState int

const (
				Playing GameState = iota
				GameOver
				Victory
)


type Game struct {
	ship         Ship
	enemyShip      *EnemyShip
				gameStartTime  time.Time
	bullets      []Bullet
	asteroids    []Asteroid
	powerUps     []PowerUp
	score        int
	highScore    int
	powerUp      PowerUpType
	powerUpEnd   time.Time
	audioContext *audio.Context
	shootSound   *audio.Player
	gameOverSound *audio.Player
	enableSound  bool
	gameOver     bool
	lastPowerUpSpawn time.Time
	asteroidSpawnRate time.Duration
	powerUpMessage string
	powerUpMessageTime time.Time
	state           GameState
				gameTime        time.Duration
				asteroidSpeed   float64
				enemyShipSpeed  float64
				asteroidsDefeated int
				enemyShipsDestroyed int
				gameFont        font.Face
}

type PowerUpType int

const (
	screenWidth     = 800
	screenHeight    = 600
	shipSpeed       = 4
	bulletSpeed     = 6
	asteroidSpeed   = 5
	maxAsteroids    = 10
	maxVertices     = 8
	shipHealth      = 100
	powerUpDuration = 30 // seconds
	sampleRate      = 44100
	powerUpSpawnInterval = 45 * time.Second
	powerUpMessageDuration = 3 * time.Second
)

func (s *Ship) Move(dx, dy float64) {
	s.X += dx
	s.Y += dy
	if s.X < 0 {
		s.X = 0
	}
	if s.X > screenWidth {
		s.X = screenWidth
	}
	if s.Y < 0 {
		s.Y = 0
	}
	if s.Y > screenHeight {
		s.Y = screenHeight
	}
}

func (s *Ship) Draw(screen *ebiten.Image) {
	x1 := s.X + s.Size*math.Cos(s.Angle)
	y1 := s.Y + s.Size*math.Sin(s.Angle)
	x2 := s.X + s.Size*math.Cos(s.Angle+2.0944)
	y2 := s.Y + s.Size*math.Sin(s.Angle+2.0944)
	x3 := s.X + s.Size*math.Cos(s.Angle+4.1888)
	y3 := s.Y + s.Size*math.Sin(s.Angle+4.1888)
	ebitenutil.DrawLine(screen, x1, y1, x2, y2, colornames.White)
	ebitenutil.DrawLine(screen, x2, y2, x3, y3, colornames.White)
	ebitenutil.DrawLine(screen, x3, y3, x1, y1, colornames.White)
}

func (e *EnemyShip) Draw(screen *ebiten.Image) {
				x1 := e.X + e.Size*math.Cos(e.Angle)
				y1 := e.Y + e.Size*math.Sin(e.Angle)
				x2 := e.X + e.Size*math.Cos(e.Angle+2.0944)
				y2 := e.Y + e.Size*math.Sin(e.Angle+2.0944)
				x3 := e.X + e.Size*math.Cos(e.Angle+4.1888)
				y3 := e.Y + e.Size*math.Sin(e.Angle+4.1888)
				ebitenutil.DrawLine(screen, x1, y1, x2, y2, colornames.Red)
				ebitenutil.DrawLine(screen, x2, y2, x3, y3, colornames.Red)
				ebitenutil.DrawLine(screen, x3, y3, x1, y1, colornames.Red)
}

func (a *Asteroid) Draw(screen *ebiten.Image) {
	for i := 0; i < a.NumVertices; i++ {
		j := (i + 1) % a.NumVertices
		ebitenutil.DrawLine(screen, a.X+a.Vertices[i][0], a.Y+a.Vertices[i][1], a.X+a.Vertices[j][0], a.Y+a.Vertices[j][1], colornames.White)
	}
}

const (
	None PowerUpType = iota
	Nuke
	DoubleDamage
	InfiniteAmmo
)

type PowerUp struct {
	X, Y     float64
	Type     PowerUpType
	Duration time.Time
}

func (g *Game) Update() error {
				g.gameTime = time.Since(g.gameStartTime)

				switch g.state {
				case Playing:
								if g.gameOver {
												if ebiten.IsKeyPressed(ebiten.KeyControl) && ebiten.IsKeyPressed(ebiten.KeyR) {
																g.reset()
																g.gameOver = false
												}
												return nil
								}

								if time.Since(g.gameStartTime) > 105*time.Second && g.enemyShip == nil {
												g.enemyShip = &EnemyShip{
																X:     float64(rand.Intn(screenWidth)),
																Y:     0,
																Angle: math.Pi / 2,
																Size:  30,
																Health: 100,
																Speed: shipSpeed * 1.5,
												}
								}

								if g.enemyShip != nil {
															// Simple AI: move towards the player
															dx := g.ship.X - g.enemyShip.X
															dy := g.ship.Y - g.enemyShip.Y
															distance := math.Sqrt(dx*dx + dy*dy)
															g.enemyShip.X += (dx / distance) * g.enemyShip.Speed
															g.enemyShip.Y += (dy / distance) * g.enemyShip.Speed
															g.enemyShip.Angle = math.Atan2(dy, dx)

															// Check for collision with player's bullets
															for i := len(g.bullets) - 1; i >= 0; i-- {
																			bullet := &g.bullets[i]
																			if g.collidesWithEnemyShip(bullet) {
																							g.enemyShip.Health -= bullet.Damage
																							g.bullets = append(g.bullets[:i], g.bullets[i+1:]...)
																							if g.enemyShip.Health <= 0 {
																											g.enemyShip = nil
																											g.score += 100 // Bonus points for destroying enemy ship
																											g.enemyShipsDestroyed++
																											break
																							}
																			}
															}
											}

								if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
												g.ship.Move(0, -shipSpeed)
												g.ship.Angle = -math.Pi / 2
								}
								if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
												g.ship.Move(0, shipSpeed)
												g.ship.Angle = math.Pi / 2
								}
								if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
												g.ship.Move(-shipSpeed, 0)
												g.ship.Angle = math.Pi
								}
								if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
												g.ship.Move(shipSpeed, 0)
												g.ship.Angle = 0
								}

								if ebiten.IsKeyPressed(ebiten.KeySpace) {
												damage := 1
												if g.powerUp == DoubleDamage {
																damage = 2
												}
												g.bullets = append(g.bullets, Bullet{X: g.ship.X, Y: g.ship.Y, Angle: g.ship.Angle, Damage: damage})
												if g.enableSound && g.shootSound != nil {
																g.shootSound.Rewind()
																g.shootSound.Play()
												}
								}

								for i := len(g.bullets) - 1; i >= 0; i-- {
												bullet := &g.bullets[i]
												bullet.X += bulletSpeed * math.Cos(bullet.Angle)
												bullet.Y += bulletSpeed * math.Sin(bullet.Angle)
												if bullet.X < 0 || bullet.X > screenWidth || bullet.Y < 0 || bullet.Y > screenHeight {
																g.bullets = append(g.bullets[:i], g.bullets[i+1:]...)
												}
								}

								for i := len(g.asteroids) - 1; i >= 0; i-- {
												asteroid := &g.asteroids[i]
												asteroid.X += asteroid.Speed * math.Cos(asteroid.Angle)
												asteroid.Y += asteroid.Speed * math.Sin(asteroid.Angle)
												if asteroid.X < -50 || asteroid.X > screenWidth+50 || asteroid.Y < -50 || asteroid.Y > screenHeight+50 {
																g.asteroids = append(g.asteroids[:i], g.asteroids[i+1:]...)
												}
								}

								for i := len(g.asteroids) - 1; i >= 0; i-- {
												asteroid := &g.asteroids[i]
												for j := len(g.bullets) - 1; j >= 0; j-- {
																bullet := &g.bullets[j]
																if g.collides(asteroid, bullet) {
																				g.bullets = append(g.bullets[:j], g.bullets[j+1:]...)
																				if asteroid.Health <= 0 {
																								if asteroid.Size > 20 {
																												g.splitAsteroid(asteroid)
																								}
																								g.asteroids = append(g.asteroids[:i], g.asteroids[i+1:]...)
																								g.score++
																								g.asteroidsDefeated++
																				}
																				break
																}
												}
								}

								for i := len(g.asteroids) - 1; i >= 0; i-- {
												asteroid := &g.asteroids[i]
												if g.collidesWithShip(asteroid) {
																g.ship.Health -= g.getDamage(asteroid)
																g.asteroids = append(g.asteroids[:i], g.asteroids[i+1:]...)
																if g.ship.Health <= 0 {
																				g.ship.Health = 0
																				g.state = GameOver
																				if g.enableSound && g.gameOverSound != nil {
																								g.gameOverSound.Rewind()
																								g.gameOverSound.Play()
																				}
																				if g.score > g.highScore {
																								g.highScore = g.score
																				}
																}
												}
								}

								if time.Since(g.powerUpEnd) > powerUpDuration*time.Second {
												g.powerUp = None
								}

								if len(g.asteroids) < maxAsteroids {
												g.generateAsteroid()
								}

								if time.Since(g.lastPowerUpSpawn) >= 30*time.Second {
												g.spawnPowerUp()
												g.lastPowerUpSpawn = time.Now()
								}

								// Check for power-up collisions
								for i := len(g.powerUps) - 1; i >= 0; i-- {
												powerUp := &g.powerUps[i]
												if g.collidesWithPowerUp(powerUp) {
																g.activatePowerUp(powerUp.Type)
																g.powerUps = append(g.powerUps[:i], g.powerUps[i+1:]...)
												}
								}

								// Increase asteroid generation speed
								g.asteroidSpawnRate = time.Duration(float64(g.asteroidSpawnRate) * 0.999)

								// Increase difficulty over time
								g.asteroidSpeed = asteroidSpeed * (1 + float64(g.gameTime.Minutes()) * 0.1)
								if g.enemyShip != nil {
												g.enemyShipSpeed = shipSpeed * (1.5 + float64(g.gameTime.Minutes()) * 0.1)
								}

								// Check for victory condition
								if g.gameTime >= 5*time.Minute {
												g.state = Victory
								}

				case GameOver, Victory:
								if ebiten.IsKeyPressed(ebiten.KeySpace) {
												g.reset()
								}
				}

				return nil
}



func (g *Game) collidesWithEnemyShip(bullet *Bullet) bool {
				if g.enemyShip == nil {
								return false
				}
				dx := bullet.X - g.enemyShip.X
				dy := bullet.Y - g.enemyShip.Y
				distance := math.Sqrt(dx*dx + dy*dy)
				return distance <= g.enemyShip.Size
}


func (g *Game) Draw(screen *ebiten.Image) {
				switch g.state {
				case Playing:
								if g.enemyShip != nil {
												g.enemyShip.Draw(screen)
								}

								g.ship.Draw(screen)

								for _, bullet := range g.bullets {
												ebitenutil.DrawRect(screen, bullet.X, bullet.Y, 2, 2, colornames.Yellow)
								}

								for _, asteroid := range g.asteroids {
												asteroid.Draw(screen)
								}

								for _, powerUp := range g.powerUps {
												color := colornames.Red
												if powerUp.Type == DoubleDamage {
																color = colornames.Blue
												} else if powerUp.Type == InfiniteAmmo {
																color = colornames.Green
												}
												ebitenutil.DrawRect(screen, powerUp.X, powerUp.Y, 20, 20, color)
								}

								// Draw health bar
								ebitenutil.DrawRect(screen, 10, screenHeight-20, float64(g.ship.Health)*2, 10, colornames.White)
								ebitenutil.DebugPrint(screen, fmt.Sprintf("Score: %d\nHigh Score: %d", g.score, g.highScore))

								// Draw power-up message
								if time.Since(g.powerUpMessageTime) < powerUpMessageDuration {
												ebitenutil.DebugPrintAt(screen, g.powerUpMessage, screenWidth-200, screenHeight-20)
								}

								if time.Since(g.powerUpMessageTime) < 3*time.Second {
												ebitenutil.DebugPrintAt(screen, g.powerUpMessage, screenWidth-200, screenHeight-20)
								}
								// Draw active power-up indicator
								if g.powerUp != None {
												ebitenutil.DrawRect(screen, screenWidth-110, screenHeight-30, 100, 20, colornames.Yellow)
												ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Power-up: %v", g.powerUp), screenWidth-105, screenHeight-25)
								}

				case GameOver:
								g.drawEndScreen(screen, "Game Over")

				case Victory:
								g.drawEndScreen(screen, "Victory!")
				}
}

func (g *Game) drawEndScreen(screen *ebiten.Image, message string) {
				ebitenutil.DrawRect(screen, 0, 0, screenWidth, screenHeight, colornames.Black)
				text.Draw(screen, message, g.gameFont, screenWidth/2-50, screenHeight/2-50, colornames.White)
				text.Draw(screen, fmt.Sprintf("Score: %d", g.score), g.gameFont, screenWidth/2-50, screenHeight/2, colornames.White)
				text.Draw(screen, fmt.Sprintf("Asteroids Defeated: %d", g.asteroidsDefeated), g.gameFont, screenWidth/2-50, screenHeight/2+50, colornames.White)
				text.Draw(screen, fmt.Sprintf("Enemy Ships Destroyed: %d", g.enemyShipsDestroyed), g.gameFont, screenWidth/2-50, screenHeight/2+100, colornames.White)
				text.Draw(screen, "Press SPACE to play again", g.gameFont, screenWidth/2-100, screenHeight-50, colornames.White)
}

func (g *Game) reset() {
	g.ship = Ship{X: screenWidth / 2, Y: screenHeight - 50, Size: 20, Health: shipHealth}
	g.bullets = nil
	g.asteroids = nil
	g.powerUps = nil
	g.score = 0
	g.powerUp = None
	g.lastPowerUpSpawn = time.Now()
	g.asteroidSpawnRate = 2 * time.Second
				g.state = Playing
				g.gameStartTime = time.Now()
				g.asteroidsDefeated = 0
				g.enemyShipsDestroyed = 0
}

func (g *Game) collides(asteroid *Asteroid, bullet *Bullet) bool {
				dx := bullet.X - asteroid.X
				dy := bullet.Y - asteroid.Y
				distance := math.Sqrt(dx*dx + dy*dy)
				if distance <= asteroid.Size {
								asteroid.Size -= 5 // Reduce size when hit
								asteroid.Health -= bullet.Damage
								if asteroid.Size < 10 {
												asteroid.Health = 0 // Destroy if too small
								}
								return true
				}
				return false
}

func (g *Game) collidesWithShip(asteroid *Asteroid) bool {
	dx := g.ship.X - asteroid.X
	dy := g.ship.Y - asteroid.Y
	distance := math.Sqrt(dx*dx + dy*dy)
	return distance <= asteroid.Size + g.ship.Size
}

func (g *Game) collidesWithPowerUp(powerUp *PowerUp) bool {
	dx := g.ship.X - powerUp.X
	dy := g.ship.Y - powerUp.Y
	distance := math.Sqrt(dx*dx + dy*dy)
	return distance <= g.ship.Size + 10 // Assuming power-up size is 20x20
}

func (g *Game) getDamage(asteroid *Asteroid) int {
	if asteroid.Size < 20 {
		return 5
	} else if asteroid.Size < 40 {
		return 15
	}
	return 25
}

func (g *Game) generateAsteroid() {
	size := float64(rand.Intn(40) + 10)
	numVertices := rand.Intn(5) + 3
	asteroid := Asteroid{
		X:           float64(rand.Intn(screenWidth)),
		Y:           -50,
		NumVertices: numVertices,
		Size:        size,
		Angle:       rand.Float64() * math.Pi * 2,
		Speed:       float64(rand.Intn(3) + 1),
		Health:      int(size),
	}
	asteroid.Vertices = make([][2]float64, numVertices)
	for i := 0; i < numVertices; i++ {
		angle := float64(i) * (2 * math.Pi / float64(numVertices))
		asteroid.Vertices[i][0] = size * math.Cos(angle)
		asteroid.Vertices[i][1] = size * math.Sin(angle)
	}
	g.asteroids = append(g.asteroids, asteroid)
}



func (g *Game) splitAsteroid(asteroid *Asteroid) {
				newSize := asteroid.Size / 2
				for i := 0; i < 2; i++ {
								newAsteroid := Asteroid{
												X:           asteroid.X,
												Y:           asteroid.Y,
												NumVertices: asteroid.NumVertices,
												Size:        newSize,
												Angle:       rand.Float64() * math.Pi * 2,
												Speed:       asteroid.Speed * 1.5,
												Health:      int(newSize),
								}
								newAsteroid.Vertices = make([][2]float64, newAsteroid.NumVertices)
								for j := 0; j < newAsteroid.NumVertices; j++ {
												angle := float64(j) * (2 * math.Pi / float64(newAsteroid.NumVertices))
												newAsteroid.Vertices[j][0] = newSize * math.Cos(angle)
												newAsteroid.Vertices[j][1] = newSize * math.Sin(angle)
								}
								g.asteroids = append(g.asteroids, newAsteroid)
				}
}
func (g *Game) spawnPowerUp() {
	powerUpType := PowerUpType(rand.Intn(3) + 1) // Random power-up type (1-3)
	powerUp := PowerUp{
		X:    float64(rand.Intn(screenWidth)),
		Y:    float64(rand.Intn(screenHeight)),
		Type: powerUpType,
	}
	g.powerUps = append(g.powerUps, powerUp)
}

func (g *Game) activatePowerUp(powerUpType PowerUpType) {
				if g.powerUp != None {
								// Remove previous power-up
								g.powerUp = None
				}
				g.powerUp = powerUpType
				g.powerUpEnd = time.Now().Add(powerUpDuration * time.Second)
				g.powerUpMessage = fmt.Sprintf("Picked up %v", powerUpType)
				g.powerUpMessageTime = time.Now()
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func NewGame() *Game {
	rand.Seed(time.Now().UnixNano())
	game := &Game{
		ship: Ship{
			X:       screenWidth / 2,
			Y:       screenHeight - 50,
			Size:    20,
			Health:  shipHealth,
		},
		enableSound: true,
		lastPowerUpSpawn: time.Now(),
		asteroidSpawnRate: 2 * time.Second,
								state: Playing,
								gameStartTime: time.Now(),
	}
	game.generateAsteroid()

	game.audioContext = audio.NewContext(sampleRate)

	shootSound, err := loadWaveFile("assets/shoot.wav", game.audioContext)
	if err == nil {
		game.shootSound, err = game.audioContext.NewPlayer(shootSound)
		if err != nil {
			log.Printf("Error creating shoot sound player: %v", err)
		}
	} else {
		log.Printf("Error loading shoot sound file: %v", err)
	}

	gameOverSound, err := loadWaveFile("assets/game-over.wav", game.audioContext)
	if err == nil {
		game.gameOverSound, err = game.audioContext.NewPlayer(gameOverSound)
		if err != nil {
			log.Printf("Error creating game over sound player: %v", err)
		}
	} else {
		log.Printf("Error loading game over sound file: %v", err)
	}

				tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
				if err != nil {
								log.Fatal(err)
				}
				game.gameFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
								Size:    24,
								DPI:     72,
								Hinting: font.HintingFull,
				})
				if err != nil {
								log.Fatal(err)
				}

	return game
}

func loadWaveFile(path string, audioContext *audio.Context) (*wav.Stream, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	stream, err := wav.Decode(audioContext, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("error decoding WAV data: %v", err)
	}

	return stream, nil
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Space Shooter")
	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
