//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	ccn = iota
	exp
	cvv
)

var (
	hotPink       = lipgloss.Color("#FF06B7")
	darkGray      = lipgloss.Color("#767676")
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)
)

// ---------- model ----------
type model struct {
	inputs  []textinput.Model
	focused int
	err     error
}

func ccnValidator(s string) error {
	if len(s) > 19 {
		return fmt.Errorf("CCN too long")
	}
	c := strings.ReplaceAll(s, " ", "")
	_, err := strconv.ParseInt(c, 10, 64)
	return err
}

func expValidator(s string) error {
	e := strings.ReplaceAll(s, "/", "")
	_, err := strconv.ParseInt(e, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid")
	}
	if len(s) >= 3 && (strings.Index(s, "/") != 2 || strings.LastIndex(s, "/") != 2) {
		return fmt.Errorf("invalid")
	}
	return nil
}

func cvvValidator(s string) error {
	_, err := strconv.ParseInt(s, 10, 64)
	return err
}

func initialModel() model {
	inp := make([]textinput.Model, 3)
	inp[ccn] = textinput.New()
	inp[ccn].Placeholder = "4505 **** **** 1234"
	inp[ccn].CharLimit = 20
	inp[ccn].Width = 30
	inp[ccn].Focus()
	inp[ccn].Prompt = ""
	inp[ccn].Validate = ccnValidator

	inp[exp] = textinput.New()
	inp[exp].Placeholder = "MM/YY"
	inp[exp].CharLimit = 5
	inp[exp].Width = 5
	inp[exp].Prompt = ""
	inp[exp].Validate = expValidator

	inp[cvv] = textinput.New()
	inp[cvv].Placeholder = "XXX"
	inp[cvv].CharLimit = 3
	inp[cvv].Width = 5
	inp[cvv].Prompt = ""
	inp[cvv].Validate = cvvValidator

	return model{inputs: inp}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.focused == len(m.inputs)-1 {
				return m, tea.Quit
			}
			m.next()
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyShiftTab, tea.KeyCtrlP:
			m.prev()
		case tea.KeyTab, tea.KeyCtrlN:
			m.next()
		}
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.inputs[m.focused].Focus()
	}
	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return fmt.Sprintf(
		` Total: $21.50

 %s
 %s

 %s  %s
 %s  %s

 %s
`,
		inputStyle.Width(30).Render("Card Number"),
		m.inputs[ccn].View(),
		inputStyle.Width(6).Render("EXP"),
		inputStyle.Width(6).Render("CVV"),
		m.inputs[exp].View(),
		m.inputs[cvv].View(),
		continueStyle.Render("Continue ->"),
	)
}
func (m *model) next() { m.focused = (m.focused + 1) % len(m.inputs) }
func (m *model) prev() {
	m.focused--
	if m.focused < 0 {
		m.focused = len(m.inputs) - 1
	}
}

// ---------- I/O bridge ----------
type MinReadBuffer struct{ buf *bytes.Buffer }

func (b *MinReadBuffer) Read(p []byte) (int, error) {
	for b.buf.Len() == 0 {
		time.Sleep(30 * time.Millisecond)
	}
	return b.buf.Read(p)
}
func (b *MinReadBuffer) Write(p []byte) (int, error) { return b.buf.Write(p) }

func createTeaForJS(m tea.Model) *tea.Program {
	fromJs := &MinReadBuffer{buf: bytes.NewBuffer(nil)}
	fromGo := bytes.NewBuffer(nil)

	prog := tea.NewProgram(m, tea.WithInput(fromJs), tea.WithOutput(fromGo), tea.WithAltScreen())

	js.Global().Set("bubbletea_write", js.FuncOf(func(this js.Value, args []js.Value) any {
		fromJs.Write([]byte(args[0].String()))
		return nil
	}))
	js.Global().Set("bubbletea_read", js.FuncOf(func(this js.Value, args []js.Value) any {
		b := make([]byte, fromGo.Len())
		_, _ = fromGo.Read(b)
		fromGo.Reset()
		return string(b)
	}))
	js.Global().Set("bubbletea_resize", js.FuncOf(func(this js.Value, args []js.Value) any {
		prog.Send(tea.WindowSizeMsg{Width: args[0].Int(), Height: args[1].Int()})
		return nil
	}))
	return prog
}

func main() {
	p := createTeaForJS(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
