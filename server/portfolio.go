package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type PortfolioSection struct {
	Command string   `json:"command"`
	Title   string   `json:"title"`
	Header  string   `json:"header"`
	Content []string `json:"content"`
}

type PortfolioManager struct {
	sections map[string]PortfolioSection
}

func NewPortfolioManager(contentDir string) (*PortfolioManager, error) {
	pm := &PortfolioManager{
		sections: make(map[string]PortfolioSection),
	}

	err := pm.loadSections(contentDir)
	if err != nil {
		return nil, err
	}

	return pm, nil
}

func (pm *PortfolioManager) loadSections(contentDir string) error {
	files, err := filepath.Glob(filepath.Join(contentDir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // skip files we can't read
		}

		var section PortfolioSection
		if err := json.Unmarshal(data, &section); err != nil {
			continue // skip invalid JSON
		}

		pm.sections[section.Command] = section
	}

	return nil
}

func (pm *PortfolioManager) GetSection(command string) (PortfolioSection, bool) {
	section, exists := pm.sections[command]
	return section, exists
}

func (pm *PortfolioManager) GetAllCommands() []string {
	commands := make([]string, 0, len(pm.sections))
	for cmd := range pm.sections {
		commands = append(commands, cmd)
	}
	return commands
}

func (pm *PortfolioManager) RenderSection(out io.Writer, section PortfolioSection) {
	// Clear screen and show header
	fmt.Fprint(out, "\033[H\033[2J")
	fmt.Fprintf(out, "%s\n", strings.Repeat("=", 60))
	fmt.Fprintf(out, "%s\n", section.Header)
	fmt.Fprintf(out, "%s\n", strings.Repeat("=", 60))
	fmt.Fprintln(out)

	// Render content
	for _, line := range section.Content {
		fmt.Fprintln(out, line)
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 60))
	fmt.Fprintln(out, "Press Enter to return to main menu...")
}
