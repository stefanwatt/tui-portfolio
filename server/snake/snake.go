package snake

import (
	"math"

	tl "github.com/JoelOtter/termloop"
)

// NewSnake will create a new snake and is called when the game is initialized.
func NewSnake() *Snake {
	snake := new(Snake)
	snake.Entity = tl.NewEntity(5, 5, 1, 1)
	snake.Direction = right
	snake.MovementCounter = 0
	snake.Speed = DEFAULT_SPEED

	snake.Bodylength = []Coordinates{
		{1, 6}, // Tail
		{2, 6}, // Body
		{3, 6}, // Head
	}

	return snake
}

// Head is the snake head which is used to move the snake around.
// The head is also the hitbox for food, border and the snake itself.
func (snake *Snake) Head() *Coordinates {
	return &snake.Bodylength[len(snake.Bodylength)-1]
}

// BorderCollision checks if the arena border contains the snakes head, if so it will return true.
func (snake *Snake) BorderCollision() bool {
	return gs.ArenaEntity.Contains(*snake.Head())
}

// FoodCollision checks if the food contains the snakes head, if so it will return true.
func (snake *Snake) FoodCollision() bool {
	return gs.FoodEntity.Contains(*snake.Head())
}

// SnakeCollision checks if the snakes body contains its head, if so it will return true.
func (snake *Snake) SnakeCollision() bool {
	return snake.Contains()
}

func isOpposite(a, b direction) bool {
	return (a == up && b == down) || (a == down && b == up) || (a == left && b == right) || (a == right && b == left)
}

// Draw will check every tick and draw the snake on the screen, it also checks if the snake has any collisions
// using the funtions above.
func (snake *Snake) Draw(screen *tl.Screen) {
	// Increment movement counter
	snake.MovementCounter++

	// Calculate movement interval based on speed
	movementInterval := int(60 / float64(snake.Speed))
	// Vertical moves should happen a bit less often to compensate for tall cells
	if snake.Direction == up || snake.Direction == down {
		movementInterval = int(math.Round(float64(movementInterval) * 1.6))
	}
	if movementInterval < 1 {
		movementInterval = 1
	}

	// Only move snake when counter reaches interval
	if snake.MovementCounter >= movementInterval {
		snake.MovementCounter = 0 // Reset counter

		// Decide how many sub-steps to process (allow tight cornering with two inputs)
		substeps := 1
		if len(snake.PendingDirections) >= 2 {
			substeps = 2
		}

		speedChanged := false

		for step := 0; step < substeps; step++ {
			// Apply at most one pending direction change before this sub-step
			if len(snake.PendingDirections) > 0 {
				nextDir := snake.PendingDirections[0]
				if !isOpposite(nextDir, snake.Direction) {
					snake.Direction = nextDir
				}
				// pop
				snake.PendingDirections = snake.PendingDirections[1:]
			}

			// Compute prospective head position
			nHead := *snake.Head()
			switch snake.Direction {
			case up:
				nHead.Y--
			case down:
				nHead.Y++
			case left:
				nHead.X--
			case right:
				nHead.X++
			}

			// Check border collision at prospective position
			if gs.ArenaEntity.Contains(nHead) {
				Gameover()
				return
			}

			// Check self-collision at prospective position
			for i := 0; i < len(snake.Bodylength)-1; i++ {
				if nHead == snake.Bodylength[i] {
					Gameover()
					return
				}
			}

			// Food collision against prospective position
			if gs.FoodEntity.Contains(nHead) {
				switch gs.FoodEntity.Emoji {
				case FAVOURITE_FOOD:
					if snake.Speed-3 <= DEFAULT_SPEED {
						snake.Speed = DEFAULT_SPEED
						UpdateScore(5)
					} else {
						snake.Speed -= 3
						UpdateScore(5)
					}
					speedChanged = true
					snake.Bodylength = append(snake.Bodylength, nHead)
				case SPEED_UP_FOOD:
					snake.Speed++
					speedChanged = true
				default:
					UpdateScore(1)
					snake.Bodylength = append(snake.Bodylength, nHead)
				}
				gs.FoodEntity.MoveFood()
			} else {
				// Normal movement - remove 1 segment from tail and add new head
				snake.Bodylength = append(snake.Bodylength[1:], nHead)
			}

			// Update position to new head for next sub-step and rendering
			snake.SetPosition(nHead.X, nHead.Y)
		}

		// If speed changed during the tick, update UI and prime the next interval
		if speedChanged {
			UpdateFPS()
			// Prime movement counter so the next movement reflects the new speed promptly
			newInterval := int(60 / float64(snake.Speed))
			if snake.Direction == up || snake.Direction == down {
				newInterval = int(math.Round(float64(newInterval) * 1.6))
			}
			if newInterval < 1 {
				newInterval = 1
			}
			// Set counter so that we are one frame away from next move at new speed
			snake.MovementCounter = newInterval - 1
		}
	}

	// Always render the snake (60 FPS rendering)
	for _, c := range snake.Bodylength {
		screen.RenderCell(c.X, c.Y, &tl.Cell{
			Bg: CheckSelectedColor(counterSnake),
			// Ch: '▯',
		})
	}
}

// Contains checks if the snake contains the head of the snake, if so it will return true.
func (snake *Snake) Contains() bool {
	// This for loop will check if the head is in any part of the body.
	for i := 0; i < len(snake.Bodylength)-1; i++ {
		// If the head is in any part of the body, it will return true.
		if *snake.Head() == snake.Bodylength[i] {
			return true
		}
	}
	// It will return false if the snake is not colliding with itself.
	return false
}
