package snake

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	tl "github.com/JoelOtter/termloop"
)

var (
	DEFAULT_SPEED = 8
	DEFAULT_FPS   = 60.0
)

// StartGame will start the game with the tilescreen.
func StartGame() {
	sg = tl.NewGame()
	gs = NewGamescreen()
	sg.Screen().SetLevel(gs)
	sg.Screen().SetFps(DEFAULT_FPS)
	sg.Start()
}

func NewGamescreen() *Gamescreen {
	// Creates the gamescreen level and create the entities
	gs = new(Gamescreen)
	gs.Level = tl.NewBaseLevel(tl.Cell{
		Bg: tl.ColorBlack,
	})
	gs.Score = 0
	gs.SnakeEntity = NewSnake()
	SetDiffiultyFPS()
	gs.ArenaEntity = NewArena(70, 25)
	gs.FoodEntity = NewFood()
	gs.SidepanelObject = NewSidepanel()

	// Add entities for the game level.
	gs.AddEntity(gs.FoodEntity)
	gs.AddEntity(gs.SidepanelObject.Background)
	gs.AddEntity(gs.SidepanelObject.ScoreText)
	gs.AddEntity(gs.SidepanelObject.SpeedText)
	gs.AddEntity(gs.SidepanelObject.DifficultyText)
	gs.AddEntity(gs.SnakeEntity)
	gs.AddEntity(gs.ArenaEntity)

	// Range over the instructions and add them to the entities
	y := 7
	for _, v := range sp.Instructions {
		var i *tl.Text
		y += 2
		i = tl.NewText(70+2, y, v, tl.ColorBlack, tl.ColorWhite)
		gs.AddEntity(i)
	}

	// Set Fps and return the screen
	sg.Screen().SetFps(gs.FPS)

	return gs
}

// NewSidepanel will create a new sidepanel given the arena height and width.
func NewSidepanel() *Sidepanel {
	// Create a sidepanel and its objects and return it
	sp = new(Sidepanel)
	sp.Instructions = []string{
		"Instructions:",
		"Use ← → ↑ ↓ to move the snake around",
		"Pick up the food to grow bigger",
		"■: 1 point/growth",
		"R: 5 points (removes some speed!)",
		"S: 1 point (increased speed!!)",
	}

	sp.Background = tl.NewRectangle(70+1, 0, 45, 25, tl.ColorWhite)
	sp.ScoreText = tl.NewText(70+2, 1, fmt.Sprintf("Score: %d", gs.Score), tl.ColorBlack, tl.ColorWhite)
	sp.SpeedText = tl.NewText(70+2, 3, fmt.Sprintf("Speed: %d", gs.SnakeEntity.Speed), tl.ColorBlack, tl.ColorWhite)
	sp.DifficultyText = tl.NewText(70+2, 5, fmt.Sprintf("Difficulty: %s", Difficulty), tl.ColorBlack, tl.ColorWhite)
	return sp
}

func Gameover() {
	// Create a new gameover screen and its content.
	gos := new(Gameoverscreen)
	gos.Level = tl.NewBaseLevel(tl.Cell{
		Bg: tl.ColorBlack,
	})
	logofile, _ := ioutil.ReadFile("util/gameover-logo.txt")
	gos.Logo = tl.NewEntityFromCanvas(10, 3, tl.CanvasFromString(string(logofile)))
	gos.Finalstats = []*tl.Text{
		tl.NewText(10, 13, fmt.Sprintf("Score: %d", gs.Score), tl.ColorWhite, tl.ColorBlack),
		tl.NewText(10, 15, fmt.Sprintf("Speed: %.0f", gs.FPS), tl.ColorWhite, tl.ColorBlack),
		tl.NewText(10, 17, fmt.Sprintf("Difficulty: %s", Difficulty), tl.ColorWhite, tl.ColorBlack),
	}
	gos.OptionsBackground = tl.NewRectangle(45, 12, 45, 7, tl.ColorWhite)
	gos.OptionsText = []*tl.Text{
		tl.NewText(47, 13, "Press \"r\" to restart!", tl.ColorBlack, tl.ColorWhite),
		tl.NewText(47, 15, "Press \"Delete\" to quit!", tl.ColorBlack, tl.ColorWhite),
	}

	// Add all of the entities to the screen
	for _, v := range gos.Finalstats {
		gos.AddEntity(v)
	}
	gos.AddEntity(gos.Logo)
	gos.AddEntity(gos.OptionsBackground)

	for _, vv := range gos.OptionsText {
		gos.AddEntity(vv)
	}

	// Set the screen
	sg.Screen().SetLevel(gos)
}

// UpdateScore updates the score with the given amount of points.
func UpdateScore(amount int) {
	gs.Score += amount
	sp.ScoreText.SetText(fmt.Sprintf("Score: %d", gs.Score))
}

// UpdateFPS updates the fps text.
func UpdateFPS() {
	sp.SpeedText.SetText(fmt.Sprintf("Speed: %d", gs.SnakeEntity.Speed))
}

// RestartGame will restart the game and reset the position of the food and the snake to prevent collision issues.
func RestartGame() {
	// Removes the current snake and food from the level.
	gs.RemoveEntity(gs.SnakeEntity)
	gs.RemoveEntity(gs.FoodEntity)

	// Generate a new snake and food.
	gs.SnakeEntity = NewSnake()
	gs.FoodEntity = NewFood()

	// Revert the score and fps to the standard.
	SetDiffiultyFPS()
	gs.Score = 0

	// Update the score and fps text.
	sp.ScoreText.SetText(fmt.Sprintf("Score: %d", gs.Score))
	sp.SpeedText.SetText(fmt.Sprintf("Speed: %d", gs.SnakeEntity.Speed))

	// Adds the snake and food back and sets them to the standard position.
	gs.AddEntity(gs.SnakeEntity)
	gs.AddEntity(gs.FoodEntity)
	sg.Screen().SetFps(gs.FPS)
	sg.Screen().SetLevel(gs)
}

func SetDiffiultyFPS() {
	gs.FPS = DEFAULT_FPS
	gs.SnakeEntity.Speed = DEFAULT_SPEED // Movement every 60/8 = 7.5 frames
}

func SaveHighScore(score int, speed float64, difficulty string) {
	var newRow []byte
	datetime := time.Now()
	newRow = []byte(fmt.Sprintf("\n|" + fmt.Sprintf("%s", datetime.Format("01-02-2006 15:04:05")) + "|" + fmt.Sprintf("%d", score) + "|" + fmt.Sprintf("%.0f", speed) + "|" + difficulty + "|  "))
	f, err := os.OpenFile("HIGHSCORES.md", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening file: %s", err)
	}

	_, err2 := f.Write(newRow)
	if err2 != nil {
		log.Fatalf("Error writing to file: %s", err2)
	}

	f.Close()
}
