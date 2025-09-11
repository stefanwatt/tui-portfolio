package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
	snake "tui-portfolio/server/snake"
	"unicode/utf8"
)

var (
	PROMPT = "[stefan.watt@portfolio]$ "
	SPLASH = `
      ////\\\\               ⠀⠀⠀⠀⠀⠀ ⢀⣠⣤⣴⣶⣶⠿⠿⠿⠿⠿⠿⢶⣶⣦⣤⣄⡀⠀⠀⠀⠀⠀⠀
      |      |                 ⠀⠀⠀⢀⣴⣾⠿⠛⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠛⠿⣷⣦⡀⠀⠀⠀
     @  O  O  @                ⠀⢀⣴⡿⠋⠀⠀⠀⠀              ⠀⠀⠙⢿⣦⡀⠀
      |  ~   |         \__     ⢠⣿⠋⠀⠀⠀⠀⠀⠀Welcome to my ⠀⠀⠀⠀⠙⣿⡄
       \ -- /          |\ |    ⣾⡏⠀⠀⠀⠀⠀⠀⠀portfolio!⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣷
     ___|  |___        | \|    ⣿⡇⠀⠀⠀⠀⠀⠀⠀Try running the⠀⠀⠀⠀⢸⣿
    /          \      /|__|    ⠸⣿⡄⠀⠀⠀⠀⠀⠀help  command. ⠀⠀⠀⢠⣿⠇
   /            \    / /       ⠀⠙⢿⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⡿⠋⠀
  /  /| .  . |\  \  / /        ⠀⠀⠀⠙⣿⡆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣶⠿⠋⠀⠀⠀
 /  / |      | \  \/ /         ⠀⠀⠀⢰⣿⠀⠀⠀⠀⠀⢀⣶⣦⣤⣤⣤⣤⣴⣶⣶⠿⠿⠛⠉⠀⠀⠀⠀⠀⠀
<  <  |      |  \   /          ⠀⠀⣠⣿⠃⠀⢀⣠⣤⣾⠟⠋⠈⠉⠉⠉⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
 \  \ |  .   |   \_/           ⠀⠀⢿⣷⡾⠿⠟⠛⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  \  \|______|
    \_|______|
      |      |
      |  |   |
      |  |   |
      |__|___|
      |  |  |
      (  (  |
      |  |  |
      |  |  |
     _|  |  |
 cccC_Cccc___)
		`
)

// typewriterWriter writes output with a small delay between characters.
// It passes ANSI escape sequences through without delay and writes newlines immediately.
type typewriterWriter struct {
	w     io.Writer
	delay time.Duration
}

func (tw *typewriterWriter) Write(p []byte) (int, error) {
	written := 0
	for i := 0; i < len(p); {
		// Pass through ANSI escape sequences quickly
		if p[i] == 0x1b { // ESC
			j := i + 1
			for j < len(p) {
				b := p[j]
				// Typical ANSI CSI ends with a letter
				if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') {
					j++
					break
				}
				j++
			}
			n, err := tw.w.Write(p[i:j])
			written += n
			if err != nil {
				return written, err
			}
			i = j
			continue
		}

		r, size := utf8.DecodeRune(p[i:])
		if r == utf8.RuneError && size == 1 {
			// write raw byte
			if _, err := tw.w.Write(p[i : i+1]); err != nil {
				return written, err
			}
			written++
			i++
			continue
		}

		chunk := p[i : i+size]
		if _, err := tw.w.Write(chunk); err != nil {
			return written, err
		}
		written += size

		if r != '\n' && r != '\r' && r != '\t' && r != ' ' {
			time.Sleep(tw.delay)
		}
		i += size
	}
	return written, nil
}

func cli(in io.Reader, out io.Writer, logger *ConsoleLogger) {
	// Initialize portfolio manager
	pm, err := NewPortfolioManager("content")
	if err != nil {
		logger.LogError("Could not load portfolio content: " + err.Error())
		fmt.Fprintf(out, "Warning: Could not load portfolio content: %v\n", err)
		pm = &PortfolioManager{sections: make(map[string]PortfolioSection)}
	} else {
		logger.LogInfo("Portfolio content loaded successfully")
		commands := pm.GetAllCommands()
		logger.LogDebug(fmt.Sprintf("Loaded %d portfolio sections: %v", len(commands), commands))
	}

	fmt.Fprintln(out, SPLASH)
	reader := bufio.NewReader(in)

	for {
		fmt.Fprint(out, PROMPT)
		var lineRunes []rune
		for {
			r, size, err := reader.ReadRune()
			if err != nil {
				logger.LogError("Error reading input: " + err.Error())
				fmt.Fprintln(out, "error:", err)
				return
			}

			// Normalize CR to LF
			if r == '\n' || r == '\r' {
				// Treat CR or LF as newline; if CR, emit CRLF for proper line break
				fmt.Fprint(out, "\r\n")
				line := string(lineRunes)
				line = strings.TrimSpace(line)

				logger.LogDebug("Command received: '" + line + "'")

				switch line {
				case "quit", "exit":
					logger.LogInfo("User requested exit")
					return
				case "credit":
					logger.LogInfo("Starting credit card example")
					return
				case "?":
					logger.LogInfo("Starting snake game")
					snake.StartGame()
				case "clear":
					logger.LogDebug("Clearing screen")
					fmt.Fprint(out, "\033[H\033[2J")
				case "help":
					logger.LogDebug("Showing help")
					// Generate dynamic help based on available portfolio sections
					fmt.Fprintln(out, "Available commands:")
					fmt.Fprintln(out, "  help      Show this help message")
					fmt.Fprintln(out, "  credit    Start the Bubble Tea credit card example")
					fmt.Fprintln(out, "  ?         󰪰 A little easter egg. What could it be?")
					fmt.Fprintln(out, "  clear     Clear the terminal")
					fmt.Fprintln(out, "  quit      Exit the application")

					// Add dynamic portfolio section commands
					commands := pm.GetAllCommands()
					if len(commands) > 0 {
						fmt.Fprintln(out, "")
						fmt.Fprintln(out, "Portfolio sections:")
						for _, cmd := range commands {
							section, _ := pm.GetSection(cmd)
							fmt.Fprintf(out, "  %-8s %s\n", cmd, section.Title)
						}
					}
				default:
					// Check if it's a portfolio section command
					logger.LogInfo("Trying to render portfolio section: " + line)
					if section, exists := pm.GetSection(line); exists {
						logger.LogInfo("Rendering portfolio section: " + line)
						// Typewriter effect only for section rendering
						tw := &typewriterWriter{w: out, delay: 8 * time.Millisecond}
						pm.RenderSection(tw, section)
						// Wait for user input to return to main menu
						reader.ReadString('\n')
						fmt.Fprint(out, "\033[H\033[2J") // Clear screen
					} else if line != "" {
						logger.LogDebug("Unknown command: " + line)
						fmt.Fprintln(out, "Unknown command:", line)
					}
				}

				_ = exec.Command("stty", "sane").Run()
				fmt.Fprint(out, "\033[0m")   // reset attributes
				fmt.Fprint(out, "\033[?25h") // show cursor

				// Flush any additional newlines that may follow (e.g., CRLF -> two \n)
				for reader.Buffered() > 0 {
					r2, _, err := reader.ReadRune()
					if err != nil {
						break
					}
					if r2 != '\n' {
						_ = reader.UnreadRune()
						break
					}
				}

				// Reset for next prompt
				lineRunes = nil
				break
			}

			if r == utf8.RuneError && size == 1 {
				continue
			}
			if r == 0x7f || r == '\b' {
				if len(lineRunes) > 0 {
					lineRunes = lineRunes[:len(lineRunes)-1]
					fmt.Fprint(out, "\b \b")
				}
				continue
			}
			if r >= 0x20 && r != 0x7f {
				lineRunes = append(lineRunes, r)
				fmt.Fprint(out, string(r))
			}
		}
	}
}
