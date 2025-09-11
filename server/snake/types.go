package snake

import tl "github.com/JoelOtter/termloop"

// Game Object Variables.
var (
	sg  *tl.Game
	sp  *Sidepanel
	gs  *Gamescreen
)

// Own created types.
type (
	direction   int
	difficulty  int
	colorobject int
)

// Game options
var (
	Difficulty  = "Normal"
	ColorObject = "Snake"
)

const (
	easy difficulty = iota
	normal
	hard
)

const (
	snake colorobject = iota
	food
	arena
)

const (
	up direction = iota
	down
	left
	right
)

type Gameoverscreen struct {
	tl.Level
	Logo              *tl.Entity
	Finalstats        []*tl.Text
	OptionsBackground *tl.Rectangle
	OptionsText       []*tl.Text
}

type Gamescreen struct {
	tl.Level
	FPS             float64
	Score           int
	SnakeEntity     *Snake
	FoodEntity      *Food
	ArenaEntity     *Arena
	SidepanelObject *Sidepanel
}

type Sidepanel struct {
	Background     *tl.Rectangle
	Instructions   []string
	ScoreText      *tl.Text
	SpeedText      *tl.Text
	DifficultyText *tl.Text
}

type Arena struct {
	*tl.Entity
	Width       int
	Height      int
	ArenaBorder map[Coordinates]int
}

type Snake struct {
	*tl.Entity
	Direction         direction
	Length            int
	Bodylength        []Coordinates
	Speed             int
	MovementCounter   int
	PendingDirections []direction
}

type Food struct {
	*tl.Entity
	Foodposition Coordinates
	Emoji        rune
}

type Coordinates struct {
	X int
	Y int
}
