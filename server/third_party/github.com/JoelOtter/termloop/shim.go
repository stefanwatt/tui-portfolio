package termloop

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Attr and colors (minimal)

type Attr int

const (
	ColorDefault Attr = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorBlue
	ColorYellow
	ColorMagenta
	ColorCyan
	ColorWhite
)

// Keys and events (minimal)

type EventType int

type Key int

const (
	EventKey EventType = iota
)

const (
	KeyArrowUp Key = iota + 1000
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyDelete
)

type Event struct {
	Type EventType
	Key  Key
	Ch   rune
}

// Cell used by RenderCell

type Cell struct {
	Fg Attr
	Bg Attr
	Ch rune
}

// Global IO configured by server

var (
	ioMu     sync.RWMutex
	ioReader io.Reader
	ioWriter io.Writer
)

func SetIO(r io.Reader, w io.Writer) {
	ioMu.Lock()
	defer ioMu.Unlock()
	ioReader = r
	ioWriter = w
}

// Entity base type and helpers used by the game code

type Entity struct {
	x      int
	y      int
	w      int
	h      int
	canvas [][]rune
}

func NewEntity(x, y, w, h int) *Entity {
	return &Entity{x: x, y: y, w: w, h: h}
}

func (e *Entity) SetPosition(x, y int) { e.x, e.y = x, y }

// Canvas helpers for game over logo

type Canvas [][]rune

func CanvasFromString(s string) Canvas {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	cv := make([][]rune, len(lines))
	for i, line := range lines {
		cv[i] = []rune(line)
	}
	return cv
}

func NewEntityFromCanvas(x, y int, cv Canvas) *Entity {
	return &Entity{x: x, y: y, canvas: cv}
}

// Text primitive

type Text struct {
	x, y   int
	fg, bg Attr
	text   string
}

func NewText(x, y int, text string, fg, bg Attr) *Text {
	return &Text{x: x, y: y, text: text, fg: fg, bg: bg}
}

func (t *Text) SetText(s string) { t.text = s }

func (t *Text) Draw(screen *Screen) {
	rx, ry := t.x, t.y
	for i, ch := range t.text {
		screen.RenderCell(rx+i, ry, &Cell{Fg: t.fg, Bg: t.bg, Ch: ch})
	}
}

// Rectangle primitive

type Rectangle struct {
	x, y int
	w, h int
	bg   Attr
}

func NewRectangle(x, y, w, h int, bg Attr) *Rectangle {
	return &Rectangle{x: x, y: y, w: w, h: h, bg: bg}
}

func (r *Rectangle) Draw(screen *Screen) {
	for yy := 0; yy < r.h; yy++ {
		for xx := 0; xx < r.w; xx++ {
			screen.RenderCell(r.x+xx, r.y+yy, &Cell{Bg: r.bg, Ch: ' '})
		}
	}
}

// Screen collects cells and flushes frames to ioWriter

type Screen struct {
	mu    sync.Mutex
	cells map[int]map[int]Cell
	game  *Game
}

func (s *Screen) RenderCell(x, y int, c *Cell) {
	if s == nil || c == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cells == nil {
		s.cells = make(map[int]map[int]Cell)
	}
	row := s.cells[y]
	if row == nil {
		row = make(map[int]Cell)
		s.cells[y] = row
	}
	row[x] = *c
}

// Level stores a list of entities. Gamescreen embeds this.

type Level struct {
	entities []any
	bg       Cell
}

func NewBaseLevel(bg Cell) Level {
	return Level{bg: bg}
}

func (l *Level) AddEntity(e any) {
	l.entities = append(l.entities, e)
	registerEntity(e)
}

func (l *Level) RemoveEntity(target any) {
	for i, e := range l.entities {
		if e == target {
			l.entities = append(l.entities[:i], l.entities[i+1:]...)
			unregisterEntity(target)
			return
		}
	}
}

// Game and loop

type Game struct {
	screen *Screen
	fps    float64
	quit   bool
}

func NewGame() *Game {
	g := &Game{fps: 60}
	s := &Screen{game: g, cells: make(map[int]map[int]Cell)}
	g.screen = s
	return g
}

func (g *Game) Screen() *Screen { return g.screen }

func (s *Screen) SetLevel(level interface { /* marker */
}) {
}

func (s *Screen) SetFps(f float64) {
	if s == nil || s.game == nil {
		return
	}
	s.game.fps = f
}

// Drawable interface matches existing Draw methods in snake code

type drawable interface{ Draw(*Screen) }

type ticker interface{ Tick(Event) }

func (e *Entity) Draw(screen *Screen) {
	// draw canvas if present
	if len(e.canvas) == 0 {
		return
	}
	for dy, row := range e.canvas {
		for dx, ch := range row {
			if ch == 0 {
				continue
			}
			screen.RenderCell(e.x+dx, e.y+dy, &Cell{Ch: ch})
		}
	}
}

// event reader parses minimal keys from ioReader

func readEvents(ch chan Event, stop <-chan struct{}) {
	ioMu.RLock()
	r := ioReader
	ioMu.RUnlock()
	if r == nil {
		return
	}
	br := bufio.NewReader(r)
	for {
		select {
		case <-stop:
			return
		default:
		}
		b, err := br.ReadByte()
		if err != nil {
			return
		}
		if b == 0x1b { // ESC
			b2, _ := br.ReadByte()
			if b2 == '[' {
				b3, _ := br.ReadByte()
				switch b3 {
				case 'A':
					ch <- Event{Type: EventKey, Key: KeyArrowUp}
				case 'B':
					ch <- Event{Type: EventKey, Key: KeyArrowDown}
				case 'C':
					ch <- Event{Type: EventKey, Key: KeyArrowRight}
				case 'D':
					ch <- Event{Type: EventKey, Key: KeyArrowLeft}
				case '3':
					// likely Delete: ESC [ 3 ~
					_, _ = br.ReadByte() // consume '~'
					ch <- Event{Type: EventKey, Key: KeyDelete}
				}
			}
			continue
		}
		// regular rune
		if b == '\n' || b == '\r' {
			continue
		}
		rn := rune(b)
		ch <- Event{Type: EventKey, Ch: rn}
	}
}

func (g *Game) Start() {
	ioMu.RLock()
	w := ioWriter
	ioMu.RUnlock()
	if w == nil {
		return
	}
	// start input reader
	stop := make(chan struct{})
	evCh := make(chan Event, 16)
	go readEvents(evCh, stop)

	// simple frame loop
	defer close(stop)
	last := time.Now()
	accum := 0.0
	sp := time.Second / 60
	for !g.quit {
		// events dispatch burst
		for {
			select {
			case ev := <-evCh:
				if ev.Key == KeyDelete {
					g.quit = true
				}
				broadcastTick(ev)
			default:
				goto afterDispatch
			}
		}
	afterDispatch:

		now := time.Now()
		accum += float64(now.Sub(last))
		last = now
		frameDur := time.Second / time.Duration(max(1, int(g.fps)))
		for accum >= float64(frameDur) {
			accum -= float64(frameDur)
			drawFrame(w, g.screen)
		}
		time.Sleep(sp / 4)
	}
	// final clear
	_, _ = w.Write([]byte("\033[0m\033[?25h"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// simple global registry of entities added to any Level

var (
	globalMu sync.RWMutex
	registry []any
)

func globalEntities() []any {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return append([]any(nil), registry...)
}

func registerEntity(e any) {
	globalMu.Lock()
	registry = append(registry, e)
	globalMu.Unlock()
}

func unregisterEntity(target any) {
	globalMu.Lock()
	for i, e := range registry {
		if e == target {
			registry = append(registry[:i], registry[i+1:]...)
			break
		}
	}
	globalMu.Unlock()
}

// dispatch key events to any entity implementing Tick(Event)

func broadcastTick(ev Event) {
	ents := globalEntities()
	for _, e := range ents {
		if t, ok := e.(interface{ Tick(Event) }); ok {
			t.Tick(ev)
		}
	}
}

// drawFrame renders all entities and flushes to writer as ANSI grid

func drawFrame(w io.Writer, screen *Screen) {
	// reset buffer
	screen.mu.Lock()
	screen.cells = make(map[int]map[int]Cell)
	screen.mu.Unlock()

	ents := globalEntities()
	for _, e := range ents {
		if d, ok := e.(drawable); ok {
			d.Draw(screen)
		}
		if en, ok := e.(*Entity); ok {
			en.Draw(screen)
		}
		if t, ok := e.(*Text); ok {
			t.Draw(screen)
		}
		if r, ok := e.(*Rectangle); ok {
			r.Draw(screen)
		}
	}

	// compute bounds
	minY, maxY := 0, 0
	minX, maxX := 0, 0
	screen.mu.Lock()
	for y, row := range screen.cells {
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
		for x := range row {
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
		}
	}
	// build frame with ANSI bg colors
	var sb strings.Builder
	sb.WriteString("\033[H\033[2J\033[?25l")
	currentBg := ColorDefault
	setBg := func(bg Attr) {
		if bg == currentBg {
			return
		}
		currentBg = bg
		if bg == ColorDefault {
			sb.WriteString("\033[49m")
			return
		}
		code := 49
		switch bg {
		case ColorBlack:
			code = 40
		case ColorRed:
			code = 41
		case ColorGreen:
			code = 42
		case ColorYellow:
			code = 43
		case ColorBlue:
			code = 44
		case ColorMagenta:
			code = 45
		case ColorCyan:
			code = 46
		case ColorWhite:
			code = 47
		default:
			code = 49
		}
		sb.WriteString(fmt.Sprintf("\033[%dm", code))
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			ch := ' '
			bg := ColorDefault
			if row, ok := screen.cells[y]; ok {
				if cell, ok2 := row[x]; ok2 {
					if cell.Ch != 0 {
						ch = cell.Ch
					}
					bg = cell.Bg
				}
			}
			setBg(bg)
			sb.WriteRune(ch)
		}
		// reset bg for newline to avoid bleed into prompt
		setBg(ColorDefault)
		sb.WriteByte('\n')
	}
	screen.mu.Unlock()
	// reset after frame
	sb.WriteString("\033[49m")
	_, _ = w.Write([]byte(sb.String()))
}
