package snake

import (
	tl "github.com/JoelOtter/termloop"
	tb "github.com/nsf/termbox-go"
)

var (
	counterSnake = 10
	counterArena = 10
)

// Tick listens for a keypress and then returns a direction for the snake.
func (snake *Snake) Tick(event tl.Event) {
	// Checks if the event is a keyevent.
	if event.Type == tl.EventKey {
		var next direction
		switch event.Key {
		// Checks if the key is a → press.
		case tl.KeyArrowRight:
			next = right
		// Checks if the key is a ← press.
		case tl.KeyArrowLeft:
			next = left
		// Checks if the key is a ↑ press.
		case tl.KeyArrowUp:
			next = up
		// Checks if the key is a ↓ press.
		case tl.KeyArrowDown:
			next = down
		default:
			return
		}

		// Determine most recent effective direction (last enqueued, else current)
		currentEffective := snake.Direction
		if len(snake.PendingDirections) > 0 {
			currentEffective = snake.PendingDirections[len(snake.PendingDirections)-1]
		}

		// Reject 180° turns
		if isOpposite(currentEffective, next) {
			return
		}

		// Filter: avoid enqueuing same-as-last enqueued
		if len(snake.PendingDirections) > 0 {
			last := snake.PendingDirections[len(snake.PendingDirections)-1]
			if last == next {
				return
			}
		}

		// Enqueue direction with small cap
		if len(snake.PendingDirections) < 4 {
			snake.PendingDirections = append(snake.PendingDirections, next)
		}
	}
}

// Tick is a method for the gameoverscreen which listens for either a restart or a quit input from the user.
func (gos *Gameoverscreen) Tick(event tl.Event) {
	if event.Type == tl.EventKey {
		if event.Ch == 'r' {
			RestartGame()
		} else if event.Key == tl.KeyDelete {
			tb.Close()
		}
	}
}

func CheckSelectedColor(c int) tl.Attr {
	switch c {
	case 10:
		return tl.ColorWhite
	case 12:
		return tl.ColorRed
	case 14:
		return tl.ColorGreen
	case 16:
		return tl.ColorBlue
	case 18:
		return tl.ColorYellow
	case 20:
		return tl.ColorMagenta
	case 22:
		return tl.ColorCyan
	default:
		return tl.ColorDefault
	}
}
