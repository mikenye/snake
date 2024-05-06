package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Embedded PNG files
var (
	//go:embed assets/snakehead.png
	pngSnakeHead []byte

	//go:embed assets/snakebody.png
	pngSnakeBody []byte

	//go:embed assets/snaketail.png
	pngSnakeTail []byte

	//go:embed assets/snakebend.png
	pngSnakeBend []byte

	//go:embed assets/snaketongue.png
	pngSnakeTongue []byte

	//go:embed assets/snakeskeletonhead.png
	pngSnakeSkeletonHead []byte

	//go:embed assets/snakeskeletonbody.png
	pngSnakeSkeletonBody []byte

	//go:embed assets/snakeskeletontail.png
	pngSnakeSkeletonTail []byte

	//go:embed assets/snakeskeletonbend.png
	pngSnakeSkeletonBend []byte

	//go:embed assets/cupcake.png
	pngCupcake []byte
)

// Tile sizes are 16x16 (except for head with tongue out)
const TILESIZE = 16

// Custom game types
type (
	// the direction of the snake
	snakeDirection uint8

	// which tile to render for a given segment
	snakeTile int

	// the state of the game (which screen/mode)
	gameState uint8
)

// Game states
const (

	// main menu
	StateMainMenu gameState = iota + 1

	// countdown before user takes control
	StateGameStart

	// in-game - user controls snake and eats cupcakes until snake bites itself
	StateInGame

	// turn snake into skeleton, no user control
	StateGameEnd

	// game-over screen
	StateGameOver
)

// Constants for snake direction
const (
	UP snakeDirection = iota + 1
	DOWN
	LEFT
	RIGHT
)

// constants for snake tiles
// format is 4 type bits + 4 rotation bits
//
// +----+----+----+----+----+----+----+----+
// | T3 | T2 | T1 | T0 | R3 | R2 | R1 | R0 |
// +----+----+----+----+----+----+----+----+
//
// Type bits:
//
//	0000 = head
//	0001 = body
//	0010 = bend
//	0100 = tail
//
// Rotation bits:
//
//	0000 = no rotation
//	0001 = left 90 deg
//	0010 = right 90 deg
//	0100 = 180 deg
const (
	// Type bitmask for Head tile
	SnakeTypeHead = 0b00000000

	// Type bitmask for Body tile
	SnakeTypeBody = 0b00010000

	// Type bitmask for Bend tile
	SnakeTypeBend = 0b00100000

	// Type bitmask for Tail tile
	SnakeTypeTail = 0b01000000

	// Rotation bitmask for no rptation
	SnakeRotationNone = 0b00000000

	// Rotation bitmask for CCW 90 deg
	SnakeRotationCCW90 = 0b00000001

	// Rotation bitmask for CW 90 deg
	SnakeRotationCW90 = 0b00000010

	// Rotation bitmask for 180 deg
	SnakeRotation180 = 0b00000100

	// Snake head, no rotation (facing up)
	SnakeHeadUp = SnakeTypeHead + SnakeRotationNone

	// Snake head, 180 deg rotation (facing down)
	SnakeHeadDown = SnakeTypeHead + SnakeRotation180

	// Snake head, -90 deg rotation (facing left)
	SnakeHeadLeft = SnakeTypeHead + SnakeRotationCCW90

	// Snake head, 90 deg rotation (racing right)
	SnakeHeadRight = SnakeTypeHead + SnakeRotationCW90

	// Snake body, no rotation (facing up)
	SnakeBodyUp = SnakeTypeBody + SnakeRotationNone

	// Snake body, 180 deg rotation (facing down)
	SnakeBodyDown = SnakeTypeBody + SnakeRotation180

	// Snake body, -90 deg rotation (facing left)
	SnakeBodyLeft = SnakeTypeBody + SnakeRotationCCW90

	// Snake body, 90 deg rotation (facing right)
	SnakeBodyRight = SnakeTypeBody + SnakeRotationCW90

	// Snake bend to connect left & down
	SnakeBendLD = SnakeTypeBend + SnakeRotationNone

	// Snake bend to connect right & up
	SnakeBendRU = SnakeTypeBend + SnakeRotation180

	// Snake bend to connect right & down
	SnakeBendRD = SnakeTypeBend + SnakeRotationCCW90

	// Snake bend to connect left & up
	SnakeBendLU = SnakeTypeBend + SnakeRotationCW90

	// Snake tail, no rotation (facing up)
	SnakeTailUp = SnakeTypeTail + SnakeRotationNone

	// Snake tail, 180 deg rotation (facing down)
	SnakeTailDown = SnakeTypeTail + SnakeRotation180

	// Snake tail, -90 deg rotation (facing left)
	SnakeTailLeft = SnakeTypeTail + SnakeRotationCCW90

	// Snake tail, 90 deg rotation (racing right)
	SnakeTailRight = SnakeTypeTail + SnakeRotationCW90
)

// Struct representing snake food
type Food struct {
	// Position of food
	x, y int
}

// struct representing each segment of the snake's body
type SnakeBodySegment struct {
	// Position of segment
	x, y int

	// Direction segment is pointing
	facing snakeDirection

	// Tile representing segment
	tile snakeTile

	// Next segment (or nil) - linked list
	next *SnakeBodySegment

	// Is the segment a skeleton?
	skeleton bool
}

// struct representing the entire body of the snake
type SnakeBody struct {

	// first segment
	Head *SnakeBodySegment

	// does the snake grow next update?
	grow bool

	// length of snake
	length int
}

// game object
type Game struct {

	// state of game
	state gameState

	// size of game board (in snake segments of size TILESIZExTILESIZE)
	width, height int

	// snake tiles

	ImgSnakeHead          *ebiten.Image
	ImgSnakeBody          *ebiten.Image
	ImgSnakeTail          *ebiten.Image
	ImgSnakeBend          *ebiten.Image
	ImgSnakeTongue        *ebiten.Image
	ImgSnakeHeadTongueOut *ebiten.Image

	// snake skeleton

	ImgSnakeSkeletonHead *ebiten.Image
	ImgSnakeSkeletonBody *ebiten.Image
	ImgSnakeSkeletonTail *ebiten.Image
	ImgSnakeSkeletonBend *ebiten.Image

	// snake object
	SnakeBody *SnakeBody

	// direction snake is facing
	SnakeDirection snakeDirection

	// food tile
	ImgFood *ebiten.Image

	// food object
	food *Food

	// movement speed stuff

	ticks            int
	ticksPerMovement int

	// score stuff

	score    int
	scoreBar *ebiten.Image

	// title screen text
	textSnake *ebiten.Image

	// start game countdown

	countDownNum   int
	countDownTicks int

	// fade to skeleton stuff

	skeleTicks           int
	skeleTicksPerSegment int

	// tongue stuff

	tongueShow     bool
	tongueTicks    int
	tongueTicksMin int
}

// draws the text SNAKE out of snakes
func (g *Game) InitTitleScreen(imgOut *ebiten.Image) {
	S := g.SpawnSnake(3, 1)
	g.SnakeMove(S, LEFT, false, false)
	g.SnakeMove(S, LEFT, false, false)
	g.SnakeAdvance(S, DOWN)
	g.SnakeAdvance(S, DOWN)
	g.SnakeAdvance(S, RIGHT)
	g.SnakeAdvance(S, RIGHT)
	g.SnakeAdvance(S, DOWN)
	g.SnakeAdvance(S, DOWN)
	g.SnakeAdvance(S, DOWN)
	g.SnakeAdvance(S, DOWN)
	g.SnakeAdvance(S, LEFT)
	g.SnakeAdvance(S, LEFT)
	g.SnakeAdvance(S, LEFT)
	g.SnakeAdvance(S, UP)
	g.SnakeAdvance(S, RIGHT)
	g.SnakeAdvance(S, RIGHT)
	g.SnakeAdvance(S, UP)
	g.SnakeAdvance(S, UP)
	g.SnakeAdvance(S, LEFT)
	g.SnakeAdvance(S, LEFT)
	g.SnakeAdvance(S, UP)
	g.SnakeAdvance(S, UP)
	g.SnakeAdvance(S, UP)
	g.SnakeAdvance(S, UP)
	g.SnakeAdvance(S, RIGHT)
	g.SnakeAdvance(S, RIGHT)
	g.SnakeAdvance(S, RIGHT)

	N := g.SpawnSnake(8, 7)
	g.SnakeMove(N, UP, false, false)
	g.SnakeMove(N, UP, false, false)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, LEFT)
	g.SnakeAdvance(N, LEFT)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, LEFT)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, UP)
	g.SnakeAdvance(N, RIGHT)
	g.SnakeAdvance(N, RIGHT)
	g.SnakeAdvance(N, RIGHT)
	g.SnakeAdvance(N, RIGHT)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)
	g.SnakeAdvance(N, DOWN)

	A := g.SpawnSnake(14, 2)
	g.SnakeMove(A, UP, false, false)
	g.SnakeMove(A, LEFT, false, false)
	g.SnakeAdvance(A, LEFT)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, LEFT)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, RIGHT)
	g.SnakeAdvance(A, RIGHT)
	g.SnakeAdvance(A, RIGHT)
	g.SnakeAdvance(A, RIGHT)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, DOWN)
	g.SnakeAdvance(A, LEFT)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, UP)
	g.SnakeAdvance(A, LEFT)

	K := g.SpawnSnake(18, 6)
	g.SnakeMove(K, DOWN, false, false)
	g.SnakeMove(K, LEFT, false, false)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, RIGHT)
	g.SnakeAdvance(K, DOWN)
	g.SnakeAdvance(K, DOWN)
	g.SnakeAdvance(K, DOWN)
	g.SnakeAdvance(K, DOWN)
	g.SnakeAdvance(K, DOWN)
	g.SnakeAdvance(K, RIGHT)
	g.SnakeAdvance(K, RIGHT)
	g.SnakeAdvance(K, DOWN)
	g.SnakeAdvance(K, DOWN)
	g.SnakeAdvance(K, RIGHT)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, LEFT)
	g.SnakeAdvance(K, LEFT)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, RIGHT)
	g.SnakeAdvance(K, RIGHT)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, UP)
	g.SnakeAdvance(K, LEFT)
	g.SnakeAdvance(K, DOWN)

	E := g.SpawnSnake(26, 3)
	g.SnakeMove(E, LEFT, false, false)
	g.SnakeMove(E, LEFT, false, false)
	g.SnakeAdvance(E, UP)
	g.SnakeAdvance(E, UP)
	g.SnakeAdvance(E, RIGHT)
	g.SnakeAdvance(E, RIGHT)
	g.SnakeAdvance(E, UP)
	g.SnakeAdvance(E, LEFT)
	g.SnakeAdvance(E, LEFT)
	g.SnakeAdvance(E, LEFT)
	g.SnakeAdvance(E, DOWN)
	g.SnakeAdvance(E, DOWN)
	g.SnakeAdvance(E, DOWN)
	g.SnakeAdvance(E, DOWN)
	g.SnakeAdvance(E, DOWN)
	g.SnakeAdvance(E, DOWN)
	g.SnakeAdvance(E, DOWN)
	g.SnakeAdvance(E, RIGHT)
	g.SnakeAdvance(E, RIGHT)
	g.SnakeAdvance(E, RIGHT)
	g.SnakeAdvance(E, UP)
	g.SnakeAdvance(E, LEFT)
	g.SnakeAdvance(E, LEFT)
	g.SnakeAdvance(E, UP)
	g.SnakeAdvance(E, UP)
	g.SnakeAdvance(E, RIGHT)
	g.SnakeAdvance(E, RIGHT)

	g.DrawSnake(S, imgOut, 0, false)
	g.DrawSnake(N, imgOut, 0, false)
	g.DrawSnake(A, imgOut, 0, false)
	g.DrawSnake(K, imgOut, 0, false)
	g.DrawSnake(E, imgOut, 0, false)
}

// works out the next position of the snake
func (g *Game) SnakeGetNextPos(SnakeBody *SnakeBody, d snakeDirection) (x, y int) {

	switch d {
	case UP:
		y = int(SnakeBody.Head.y) - 1
		x = int(SnakeBody.Head.x)
	case DOWN:
		y = int(SnakeBody.Head.y) + 1
		x = int(SnakeBody.Head.x)
	case LEFT:
		y = int(SnakeBody.Head.y)
		x = int(SnakeBody.Head.x) - 1
	case RIGHT:
		y = int(SnakeBody.Head.y)
		x = int(SnakeBody.Head.x) + 1
	}

	// bounds checking - wrap around the screen if needed
	if x < 0 {
		x = g.width - 1
	}
	if x > g.width-1 {
		x = 0
	}
	if y < 0 {
		y = g.height - 1
	}
	if y > g.height-1 {
		y = 0
	}

	return x, y
}

// check to see if the head of the snake is on the food tile
func (g *Game) SnakeCheckFood(SnakeBody *SnakeBody) bool {
	// check if snake just ate food
	seg := SnakeBody.Head
	if seg.x == g.food.x && seg.y == g.food.y {
		g.score++
		SnakeBody.grow = true
		return true
	}
	return false
}

// check to see if the head of the snake will eat the snake body
func (g *Game) SnakeCheckDeath(SnakeBody *SnakeBody, d snakeDirection) bool {
	// check if snake has eaten itself
	x, y := g.SnakeGetNextPos(SnakeBody, d)
	seg := g.SnakeBody.Head.next
	for {
		if seg.x == x && seg.y == y {
			return true
		}
		if seg.next == nil {
			break
		}
		seg = seg.next
	}
	return false
}

// delete tail segment
func (g *Game) SnakeRemoveTail(SnakeBody *SnakeBody) {
	// set second last segment's 'next' to nil
	prevSeg := SnakeBody.Head
	seg := SnakeBody.Head
	for i := 1; i < SnakeBody.length-1; i++ {
		prevSeg = seg
		seg = seg.next
	}
	seg.next = nil
	SnakeBody.length--

	// set tail direction
	switch {
	case prevSeg.x < seg.x:
		if math.Abs(float64(prevSeg.x-seg.x)) == 1 {
			seg.tile = SnakeTailLeft
		} else {
			seg.tile = SnakeTailRight
		}
	case prevSeg.x > seg.x:
		if math.Abs(float64(prevSeg.x-seg.x)) == 1 {
			seg.tile = SnakeTailRight
		} else {
			seg.tile = SnakeTailLeft
		}
	case prevSeg.y < seg.y:
		if math.Abs(float64(prevSeg.y-seg.y)) == 1 {
			seg.tile = SnakeTailUp
		} else {
			seg.tile = SnakeTailDown
		}
	case prevSeg.y > seg.y:
		if math.Abs(float64(prevSeg.y-seg.y)) == 1 {
			seg.tile = SnakeTailDown
		} else {
			seg.tile = SnakeTailUp
		}
	}
}

// move snake forward one segment in direction d
// will check for death condition if checkDeath is true
// will eat food if checkFood is true
func (g *Game) SnakeMove(SnakeBody *SnakeBody, d snakeDirection, checkDeath, checkFood bool) {
	// remove old tail segment if not growing
	if checkDeath {
		if g.SnakeCheckDeath(SnakeBody, d) {
			g.ChangeState(StateGameEnd)
			return
		}
	}
	if !SnakeBody.grow {
		g.SnakeRemoveTail(SnakeBody)
		g.SnakeAdvance(SnakeBody, d)
	} else {
		g.SnakeAdvance(SnakeBody, d)
		SnakeBody.grow = false
	}
	if checkFood {
		if g.SnakeCheckFood(SnakeBody) {
			SnakeBody.grow = true
			g.SpawnFood()
		}
	}
}

// advance the snake in direction d by adding a new head piece
func (g *Game) SnakeAdvance(SnakeBody *SnakeBody, d snakeDirection) {

	var headTile snakeTile

	// determine:
	//  - tile for previous segment
	//  - tile for head segment
	switch d {
	case UP:
		headTile = SnakeHeadUp
		switch SnakeBody.Head.facing {
		case LEFT:
			SnakeBody.Head.tile = SnakeBendRU
		case RIGHT:
			SnakeBody.Head.tile = SnakeBendLU
		default:
			SnakeBody.Head.tile = SnakeBodyUp
		}
	case DOWN:
		headTile = SnakeHeadDown
		switch SnakeBody.Head.facing {
		case LEFT:
			SnakeBody.Head.tile = SnakeBendRD
		case RIGHT:
			SnakeBody.Head.tile = SnakeBendLD
		default:
			SnakeBody.Head.tile = SnakeBodyDown
		}
	case LEFT:
		headTile = SnakeHeadLeft
		switch SnakeBody.Head.facing {
		case UP:
			SnakeBody.Head.tile = SnakeBendLD
		case DOWN:
			SnakeBody.Head.tile = SnakeBendLU
		default:
			SnakeBody.Head.tile = SnakeBodyLeft
		}
	case RIGHT:
		headTile = SnakeHeadRight
		switch SnakeBody.Head.facing {
		case UP:
			SnakeBody.Head.tile = SnakeBendRD
		case DOWN:
			SnakeBody.Head.tile = SnakeBendRU
		default:
			SnakeBody.Head.tile = SnakeBodyRight
		}
	}

	// update x/y coords based on direction of snake travel
	x, y := g.SnakeGetNextPos(SnakeBody, d)
	// create new head segment
	seg := SnakeBodySegment{
		x:      x,
		y:      y,
		facing: d,
		next:   SnakeBody.Head,
		tile:   headTile,
	}

	// set head as new segment
	SnakeBody.Head = &seg
	SnakeBody.length++
}

// Create a new snake
func (g *Game) SpawnSnake(startXPos, startYPos int) *SnakeBody {

	// create initial segments (3 segments, facing up)
	segTail := SnakeBodySegment{
		x:      startXPos,
		y:      startYPos + 2,
		facing: UP,
		next:   nil,
		tile:   SnakeTailUp,
	}
	segMiddle := SnakeBodySegment{
		x:      startXPos,
		y:      startYPos + 1,
		facing: UP,
		next:   &segTail,
		tile:   SnakeBodyUp,
	}
	segHead := SnakeBodySegment{
		x:      startXPos,
		y:      startYPos,
		facing: UP,
		next:   &segMiddle,
		tile:   SnakeHeadUp,
	}

	// create body
	sb := SnakeBody{
		Head:   &segHead,
		length: 3,
	}

	return &sb
}

// spawn the food tile at a random position not occupied by the snake
func (g *Game) SpawnFood() {
	var x int
	var y int
	var taken bool
	for {
		// generate a random position
		x = rand.Intn(g.width - 1)
		y = rand.Intn(g.height - 1)

		// check to see if position is taken
		taken = false
		seg := g.SnakeBody.Head
		for {
			if x == seg.x && y == seg.y {
				taken = true
				break
			}
			if seg.next == nil {
				break
			}
			seg = seg.next
		}
		if !taken {
			break
		}
	}
	f := Food{
		x: x,
		y: y,
	}
	g.food = &f
}

// draw the food tile, offsetting by yOffset (for score bar)
func (g *Game) DrawFood(imgOut *ebiten.Image, yOffset int, dimmed bool) {
	op := ebiten.DrawImageOptions{}
	xpos := g.food.x * g.ImgFood.Bounds().Dx()
	ypos := g.food.y * g.ImgFood.Bounds().Dy()

	// translate
	op.GeoM.Translate(float64(xpos), float64(ypos+yOffset))

	// if game over, fade slightly
	if dimmed {
		op.ColorScale.ScaleAlpha(0.5)
	}
	imgOut.DrawImage(g.ImgFood, &op)
}

// draw the snake, offsetting by yOffset (for score bar)
func (g *Game) DrawSnake(SnakeBody *SnakeBody, imgOut *ebiten.Image, yOffset int, dimmed bool) {
	var img *ebiten.Image
	var rotation float64
	op := ebiten.DrawImageOptions{}
	seg := SnakeBody.Head
	for {
		op.GeoM.Reset()
		op.ColorScale.Reset()

		// tile type (mask tile type bits with bitwise AND)
		switch seg.tile & 0b11110000 {
		case SnakeTypeHead:
			if seg.skeleton {
				img = g.ImgSnakeSkeletonHead
			} else {
				if g.tongueShow {
					img = g.ImgSnakeHeadTongueOut
					// as tongue out tile is 32 px high, we need to move up by 16px so the rest of the transforms/rotations work as expected
					op.GeoM.Translate(0, -TILESIZE)
				} else {
					img = g.ImgSnakeHead
				}
			}

		case SnakeTypeBody:
			if seg.skeleton {
				img = g.ImgSnakeSkeletonBody
			} else {
				img = g.ImgSnakeBody
			}

		case SnakeTypeBend:
			if seg.skeleton {
				img = g.ImgSnakeSkeletonBend
			} else {
				img = g.ImgSnakeBend
			}

		case SnakeTypeTail:
			if seg.skeleton {
				img = g.ImgSnakeSkeletonTail
			} else {
				img = g.ImgSnakeTail
			}
		default:
			panic(seg.tile)
		}

		// rotation (mask rotation bits with bitwise AND)
		switch seg.tile & 0b00001111 {
		case SnakeRotationNone:
			rotation = 0
		case SnakeRotationCCW90:
			rotation = -90 * math.Pi / 180
		case SnakeRotationCW90:
			rotation = 90 * math.Pi / 180
		case SnakeRotation180:
			rotation = math.Pi
		default:
			panic(seg.tile)
		}

		// get pixel position for segment
		xpos := seg.x * TILESIZE
		ypos := seg.y * TILESIZE

		// rotate
		if rotation != 0 {
			RotateTile(img, &op, rotation)
		}

		// translate
		op.GeoM.Translate(float64(xpos), float64(ypos+yOffset))

		// if game over, fade slightly
		if dimmed {
			op.ColorScale.ScaleAlpha(0.5)
		}

		// draw tile
		imgOut.DrawImage(img, &op)

		// if we've reached the last segment, bail out
		if seg.next == nil {
			break
		}

		// get next segment for next iteration
		seg = seg.next
	}
}

// update function for when in game
func (g *Game) UpdateInGame() error {

	// handle input
	switch {
	case ebiten.IsKeyPressed(ebiten.KeyArrowUp):
		if g.SnakeBody.Head.facing == LEFT || g.SnakeBody.Head.facing == RIGHT {
			g.SnakeDirection = UP
		}
	case ebiten.IsKeyPressed(ebiten.KeyArrowDown):
		if g.SnakeBody.Head.facing == LEFT || g.SnakeBody.Head.facing == RIGHT {
			g.SnakeDirection = DOWN
		}
	case ebiten.IsKeyPressed(ebiten.KeyArrowLeft):
		if g.SnakeBody.Head.facing == UP || g.SnakeBody.Head.facing == DOWN {
			g.SnakeDirection = LEFT
		}
	case ebiten.IsKeyPressed(ebiten.KeyArrowRight):
		if g.SnakeBody.Head.facing == UP || g.SnakeBody.Head.facing == DOWN {
			g.SnakeDirection = RIGHT
		}
	}

	// movement speed
	g.ticks++
	if g.ticks >= g.ticksPerMovement {
		g.ticks = 0
		g.SnakeMove(g.SnakeBody, g.SnakeDirection, true, true)
		g.ticksPerMovement = 40 - int(math.Min(float64(g.score), 33))
	}

	// random snake tongue
	g.RandomSnakeTongue()

	return nil
}

// random snake tongue
func (g *Game) RandomSnakeTongue() {
	g.tongueTicks++
	if g.tongueTicks >= g.tongueTicksMin {
		if g.tongueShow {
			g.tongueShow = false
		} else {
			if rand.Intn(10000) > 7000 {
				g.tongueShow = true
			}
		}
		g.tongueTicks = 0
	}
}

// update function for when in game over state
func (g *Game) UpdateGameOver() error {
	// handle input
	switch {
	case ebiten.IsKeyPressed(ebiten.KeySpace):
		g.ChangeState(StateGameStart)
	case ebiten.IsKeyPressed(ebiten.KeyEscape):
		g.ChangeState(StateMainMenu)
	}
	return nil
}

// update function for when in end game state
func (g *Game) UpdateEndGame() error {
	finished := true

	// turn snake into skeleton, one segment at a time
	g.skeleTicks++
	if g.skeleTicks >= g.skeleTicksPerSegment {
		g.skeleTicks = 0
		seg := g.SnakeBody.Head
		for {
			if !seg.skeleton {
				seg.skeleton = true
				finished = false
				break
			}
			if seg.next == nil {
				break
			}
			seg = seg.next
		}
		// if all segments are skeleton advance to game over state
		if finished {
			g.ChangeState(StateGameOver)
		}
	}
	return nil
}

// update main menu, move random background snake
func (g *Game) UpdateMainMenu() error {

	// press space to start
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.ChangeState(StateGameStart)
	}

	// movement speed & random direction
	g.ticks++
	if g.ticks >= g.ticksPerMovement {
		g.ticks = 0
		g.SnakeMove(g.SnakeBody, RandomSnakeDirection(g.SnakeBody.Head.facing), false, false)
	}

	// random snake tongue
	g.RandomSnakeTongue()

	return nil
}

// update function, ebiten calls this every tick (60 times per second)
func (g *Game) Update() error {
	var err error

	// Press Q to quit regardless of state
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		return errors.New("Q pressed")
	}

	switch g.state {

	// main menu
	case StateMainMenu:
		err = g.UpdateMainMenu()

	// game start (countdown)
	case StateGameStart:
		// count down to game start
		g.countDownTicks++
		if g.countDownTicks >= 60 {
			g.countDownNum--
			g.countDownTicks = 0
		}
		// if countdown finished, progress to in-game
		if g.countDownNum < 0 {
			g.ChangeState(StateInGame)
		}

	// in-game (user controls snake)
	case StateInGame:
		err = g.UpdateInGame()

	// end game (turn to skeleton)
	case StateGameEnd:
		err = g.UpdateEndGame()

	// game over (game over screen)
	case StateGameOver:
		err = g.UpdateGameOver()
	}

	return err
}

// draw the score bar at the top of the screen
func (g *Game) DrawScoreBar(imgOut *ebiten.Image) {
	imgOut.DrawImage(g.scoreBar, &ebiten.DrawImageOptions{})
	txt := fmt.Sprintf("Calories: %d", g.score*200)
	ebitenutil.DebugPrintAt(imgOut, txt, (g.width*TILESIZE)/2-(len(txt)*6)/2, 0)
}

// draw the main menu
func (g *Game) DrawMainMenu(imgOut *ebiten.Image) {
	g.DrawSnake(g.SnakeBody, imgOut, 16, true)
	op := ebiten.DrawImageOptions{}
	op.ColorScale.Scale(0.7, 1.5, 2, 1)
	imgOut.DrawImage(g.textSnake, &op)
	txt := "UP/DOWN/LEFT/RIGHT: Change direction of snake"
	ebitenutil.DebugPrintAt(imgOut, txt, (g.width*TILESIZE)/2-(len(txt)*6)/2, 180)
	txt = "Q: Quit"
	ebitenutil.DebugPrintAt(imgOut, txt, (g.width*TILESIZE)/2-(len(txt)*6)/2-12, 195)
	txt = "SPACE: Start Game"
	ebitenutil.DebugPrintAt(imgOut, txt, ((g.width*TILESIZE)/2-(len(txt)*6)/2)-6, 210)
	txt = "Eat the cupcakes, but not yourself!"
	ebitenutil.DebugPrintAt(imgOut, txt, (g.width*TILESIZE)/2-(len(txt)*6)/2, 265)
	txt = "github.com/mikenye/snake"
	ebitenutil.DebugPrintAt(imgOut, txt, (g.width*TILESIZE)/2-(len(txt)*6)/2, g.height*TILESIZE)
}

// draw the game over screen
func (g *Game) DrawGameOverScreen(imgOut *ebiten.Image) {
	txt := "GAME OVER!"
	ebitenutil.DebugPrintAt(imgOut, txt, (g.width*TILESIZE)/2-(len(txt)*6)/2, 130)
	txt = "SPACE: New Game"
	ebitenutil.DebugPrintAt(imgOut, txt, (g.width*TILESIZE)/2-(len(txt)*6)/2, 195)
	txt = "ESC: Main Menu"
	ebitenutil.DebugPrintAt(imgOut, txt, (g.width*TILESIZE)/2-(len(txt)*6)/2+9, 210)
	txt = "Q: Quit"
	ebitenutil.DebugPrintAt(imgOut, txt, (g.width*TILESIZE)/2-(len(txt)*6)/2, 225)
}

// draw function, ebiten calls this every tick to render the screen
func (g *Game) Draw(screen *ebiten.Image) {

	switch g.state {

	// main menu: draw the title screen
	case StateMainMenu:
		g.DrawMainMenu(screen)

	// game start: draw the game screen with countdown overlay
	case StateGameStart:
		g.DrawFood(screen, 15, false)
		g.DrawSnake(g.SnakeBody, screen, 15, false)
		if g.countDownNum > 0 {
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", g.countDownNum), 212, 135)
		} else {
			ebitenutil.DebugPrintAt(screen, "GO!", 206, 135)
		}
		g.DrawScoreBar(screen)

	// in game: draw the game screen
	case StateInGame:
		g.DrawFood(screen, 15, false)
		g.DrawSnake(g.SnakeBody, screen, 15, false)
		g.DrawScoreBar(screen)

	// in game: draw the game screen
	case StateGameEnd:
		g.DrawFood(screen, 15, false)
		g.DrawSnake(g.SnakeBody, screen, 15, false)
		g.DrawScoreBar(screen)

	// in game: draw the game screen with game over overlay
	case StateGameOver:
		g.DrawFood(screen, 15, true)
		g.DrawSnake(g.SnakeBody, screen, 15, true)
		g.DrawScoreBar(screen)
		g.DrawGameOverScreen(screen)
	}
}

// layout function, called by Ebiten to size window & content
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	screenWidth, screenHeight := g.ScreenSize()
	return screenWidth, screenHeight
}

// load images, called once on startup
func (g *Game) LoadImages() error {
	var (
		r   *bytes.Reader
		i   image.Image
		err error
	)

	// snake head
	r = bytes.NewReader(pngSnakeHead)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgSnakeHead = ebiten.NewImageFromImage(i)

	// snake skeleton head
	r = bytes.NewReader(pngSnakeSkeletonHead)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgSnakeSkeletonHead = ebiten.NewImageFromImage(i)

	// snake body
	r = bytes.NewReader(pngSnakeBody)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgSnakeBody = ebiten.NewImageFromImage(i)

	// snake skeleton body
	r = bytes.NewReader(pngSnakeSkeletonBody)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgSnakeSkeletonBody = ebiten.NewImageFromImage(i)

	// snake tail
	r = bytes.NewReader(pngSnakeTail)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgSnakeTail = ebiten.NewImageFromImage(i)

	// snake skeleton tail
	r = bytes.NewReader(pngSnakeSkeletonTail)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgSnakeSkeletonTail = ebiten.NewImageFromImage(i)

	// snake bend
	r = bytes.NewReader(pngSnakeBend)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgSnakeBend = ebiten.NewImageFromImage(i)

	// snake skeleton bend
	r = bytes.NewReader(pngSnakeSkeletonBend)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgSnakeSkeletonBend = ebiten.NewImageFromImage(i)

	// food
	r = bytes.NewReader(pngCupcake)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgFood = ebiten.NewImageFromImage(i)

	// snake tongue
	r = bytes.NewReader(pngSnakeTongue)
	i, _, err = image.Decode(r)
	if err != nil {
		return err
	}
	g.ImgSnakeTongue = ebiten.NewImageFromImage(i)

	// snake head tongue out (only tile not 16x16 - this is 16x32)
	g.ImgSnakeHeadTongueOut = ebiten.NewImage(TILESIZE, 2*TILESIZE)
	op := ebiten.DrawImageOptions{}
	// draw tongue
	g.ImgSnakeHeadTongueOut.DrawImage(g.ImgSnakeTongue, &op)
	// draw head below tongue
	op.GeoM.Translate(0, TILESIZE)
	g.ImgSnakeHeadTongueOut.DrawImage(g.ImgSnakeHead, &op)

	return nil
}

// return the size of the screen in pixels based on game width/height in tile spaces
func (g *Game) ScreenSize() (w, h int) {
	w = TILESIZE * g.width
	h = TILESIZE * g.height
	h += g.scoreBar.Bounds().Dy()
	return w, h
}

func (g *Game) ChangeState(s gameState) {
	switch s {
	case StateMainMenu:
		g.Reset()

		// grow a random snake
		for i := 0; i <= rand.Intn(100); i++ {
			g.SnakeAdvance(g.SnakeBody, RandomSnakeDirection(g.SnakeBody.Head.facing))
		}

		// fast movement speed for background snake
		g.ticksPerMovement = 5

	case StateGameStart:
		g.Reset()
	case StateInGame:
	case StateGameEnd:
	case StateGameOver:
	}
	g.state = s
}

// set initial game state
func (g *Game) Reset() {

	g.ticksPerMovement = 30
	g.SnakeDirection = UP
	g.countDownNum = 3
	g.score = 0
	g.skeleTicks = 0

	// init fresh snake body
	g.SnakeBody = g.SpawnSnake(g.width/2, g.height/2)

	// init food
	g.SpawnFood()

}

// create a new game object
func NewGame(width, height int) (*Game, error) {
	g := Game{
		width:                width,
		height:               height,
		skeleTicksPerSegment: 2,
		tongueTicksMin:       20,
	}

	// load images
	err := g.LoadImages()

	// init score bar
	g.scoreBar = ebiten.NewImage(g.width*g.ImgSnakeHead.Bounds().Dy(), 16)
	g.scoreBar.Fill(color.RGBA{34, 32, 52, 255})

	// init title screen
	w, h := g.ScreenSize()
	g.textSnake = ebiten.NewImage(w, h)
	g.InitTitleScreen(g.textSnake)

	// set initial game state
	g.Reset()
	g.ChangeState(StateMainMenu)

	return &g, err
}

// return a random direction for the snake
func RandomSnakeDirection(currentDirection snakeDirection) (d snakeDirection) {
	// should we change direction?
	if rand.Intn(100) <= 50 {
		return currentDirection
	}
	for {
		d = snakeDirection(rand.Intn(3) + 1)
		switch currentDirection {
		case UP, DOWN:
			if d == LEFT || d == RIGHT {
				return d
			}
		case LEFT, RIGHT:
			if d == UP || d == DOWN {
				return d
			}
		}
	}
}

// rotates a tile around its centre
func RotateTile(img *ebiten.Image, op *ebiten.DrawImageOptions, rotation float64) {
	op.GeoM.Translate(-TILESIZE/2, -TILESIZE/2)
	op.GeoM.Rotate(rotation)
	op.GeoM.Translate(TILESIZE/2, TILESIZE/2)
}

// main function
func main() {
	var err error

	// create new game object
	g, err := NewGame(27, 20)
	if err != nil {
		log.Fatal(err)
	}

	// set up game window
	screenWidth, screenHeight := g.ScreenSize()
	ebiten.SetWindowSize(screenWidth*2, (screenHeight * 2))
	ebiten.SetWindowTitle("Snake")

	// start game
	err = ebiten.RunGame(g)
	if err != nil {
		log.Fatal(err)
	}

}
