package main

import (
	"github.com/charmbracelet/lipgloss"
)

const (
	nostrPurple = "#8B5CF6"
	nostrPurpleDim = "#1E1B2E"
)

var tuiStyle = struct {
	Screen lipgloss.Style
	Base   lipgloss.Style
	Cursor lipgloss.Style
}{
	Screen: lipgloss.NewStyle().Background(lipgloss.Color("#000000")).Padding(0, 1),
	Base:   lipgloss.NewStyle().Background(lipgloss.Color("#000000")).Foreground(lipgloss.Color(nostrPurple)),
	Cursor: lipgloss.NewStyle().Background(lipgloss.Color(nostrPurpleDim)).Foreground(lipgloss.Color(nostrPurple)),
}

const gostrTitle = "    ________  ________  ________  _________  ________     \n   /  _____/ /  __  __\\/   ____/ /___   ___/|   __   \\    \n  /   \\  ___/  /  /  / \\____   \\    /  /    |  |__/  /    \n  \\    \\_\\  \\  \\__/  / /       /   /  /     |      _/     \n   \\________/\\______/ /_______/   /__/      |__|\\__\\      \n                                                          \n         --- NOSTR LIKE GOPHER | GOSTR ---"
