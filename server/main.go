package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tl "github.com/JoelOtter/termloop"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

// Buffer with blocking reads similar to wasm MinReadBuffer
// Provides minimal backpressure and simple API for tea.WithInput
// and for writing from WebSocket
// Not safe for concurrent writes; guarded by single goroutine usage here

type MinReadBuffer struct{ buf *bytes.Buffer }

func (b *MinReadBuffer) Read(p []byte) (int, error) {
	for b.buf.Len() == 0 {
		time.Sleep(30 * time.Millisecond)
	}
	return b.buf.Read(p)
}
func (b *MinReadBuffer) Write(p []byte) (int, error) { return b.buf.Write(p) }

// Define the same interface as src/lib/wasm/main_wasm.go expects

type ioWriter interface{ Write([]byte) (int, error) }

type ioReader interface{ Read([]byte) (int, error) }

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type resizeMsg struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

type consoleLogMsg struct {
	Type    string `json:"type"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// ConsoleLogger wraps a WebSocket connection to send console log messages
type ConsoleLogger struct {
	conn *websocket.Conn
}

func (cl *ConsoleLogger) Log(level, message string) {
	if cl.conn == nil {
		return
	}

	msg := consoleLogMsg{
		Type:    "console",
		Level:   level,
		Message: message,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	cl.conn.WriteMessage(websocket.TextMessage, data)
}

func (cl *ConsoleLogger) LogInfo(message string) {
	cl.Log("info", message)
}

func (cl *ConsoleLogger) LogError(message string) {
	cl.Log("error", message)
}

func (cl *ConsoleLogger) LogDebug(message string) {
	cl.Log("debug", message)
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	// Create console logger
	consoleLogger := &ConsoleLogger{conn: conn}
	consoleLogger.LogInfo("WebSocket connection established")

	fromClient := &MinReadBuffer{buf: bytes.NewBuffer(nil)}
	toClient := bytes.NewBuffer(nil)

	// Wire termloop shim IO so the snake game renders to WebSocket
	tl.SetIO(fromClient, toClient)

	// Output pump → send as text frames
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ticker := time.NewTicker(40 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if toClient.Len() == 0 {
					continue
				}
				b := make([]byte, toClient.Len())
				_, _ = toClient.Read(b)
				_ = conn.WriteMessage(websocket.TextMessage, b)
			}
		}
	}()

	// Reader pump → keystrokes and resize
	resizeQueue := make(chan tea.WindowSizeMsg, 4)
	go func() {
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				cancel()
				return
			}
			if msgType == websocket.TextMessage {
				// try parse resize
				var rm resizeMsg
				if err := json.Unmarshal(data, &rm); err == nil && rm.Type == "resize" {
					select {
					case resizeQueue <- tea.WindowSizeMsg{Width: rm.Cols, Height: rm.Rows}:
					default:
					}
					continue
				}
			}
			// translate CR to LF so both CLI and Bubble Tea see newlines
			for i := range data {
				if data[i] == '\r' {
					data[i] = '\n'
				}
			}
			fromClient.Write(data)
		}
	}()

	// Run CLI first
	cli(fromClient, toClient, consoleLogger)

	// Start Bubble Tea
	p := tea.NewProgram(initialModel(), tea.WithInput(fromClient), tea.WithOutput(toClient), tea.WithAltScreen())

	// Drain any queued resizes once program is ready, and forward subsequent ones
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case rs := <-resizeQueue:
				p.Send(rs)
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		toClient.Write([]byte("error: "))
		toClient.Write([]byte(err.Error()))
	}

	cancel()
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleWS)

	srv := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		log.Println("server listening on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
