package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"termcity/internal/history"
	"termcity/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	store, err := history.Open(historyDBPath())
	if err != nil {
		log.Printf("Warning: could not open history database: %v", err)
	}
	if store != nil {
		defer store.Close()
	}

	p := tea.NewProgram(
		model.NewAppModel(store),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func historyDBPath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	dir := filepath.Join(cacheDir, "termcity")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "history.db")
}
