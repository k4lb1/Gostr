package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)

const (
	menuItemHome = iota
	menuItemHomeComments
	menuItemAether
	menuItemComposeNote
	menuItemInbox
	menuItemFollowing
	menuItemFollow
	menuItemOptions
	menuItemQuit
	menuItemCount
)

const (
	optItemRelays = iota
	optItemSetKey
	optItemCount
)

var menuItems = []struct {
	prefix string
	label  string
}{
	{"1", " Home"},
	{"2", " Notes + Comments"},
	{"3", " Aether"},
	{"4", " Publish note"},
	{"5", " Inbox"},
	{"6", " Following"},
	{"7", " Follow"},
	{"8", " Optionen"},
}

var menuItemQuitLabel = "9  Quit"

var optItems = []struct {
	prefix string
	label  string
}{
	{"1", " Relays"},
	{"2", " Set key (nsec)"},
}

func updateMenu(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		if key >= "1" && key <= "9" {
			idx := int(key[0] - '1')
			if idx < menuItemCount {
				m.menuCur = idx
				return runMenuAction(m, idx)
			}
		}
		switch key {
		case "up", "k":
			m.menuCur--
			if m.menuCur < 0 {
				m.menuCur = menuItemCount - 1
			}
			return m, nil
		case "down", "j":
			m.menuCur++
			if m.menuCur >= menuItemCount {
				m.menuCur = 0
			}
			return m, nil
		case "enter", " ":
			return runMenuAction(m, m.menuCur)
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func runMenuAction(m model, idx int) (tea.Model, tea.Cmd) {
	switch idx {
	case menuItemHome:
		m.screen = screenList
		m.loading = true
		m.inbox = false
		m.notesOnly = true
		m.aether = false
		m.events = nil
		return m, loadFeedCmd(false, true, false)
	case menuItemHomeComments:
		m.screen = screenList
		m.loading = true
		m.inbox = false
		m.notesOnly = false
		m.aether = false
		m.events = nil
		return m, loadFeedCmd(false, false, false)
	case menuItemAether:
		m.screen = screenList
		m.loading = true
		m.inbox = false
		m.notesOnly = false
		m.aether = true
		m.events = nil
		return m, loadFeedCmd(false, false, true)
	case menuItemComposeNote:
		m.screen = screenComposeNote
		m.composeInput.Reset()
		m.composeInput.Focus()
		m.err = ""
		return m, textinput.Blink
	case menuItemInbox:
		m.screen = screenList
		m.loading = true
		m.inbox = true
		m.aether = false
		m.events = nil
		return m, loadFeedCmd(true, false, false)
	case menuItemFollowing:
		m.screen = screenFollowing
		m.followLines = buildFollowLines()
		return m, nil
	case menuItemFollow:
		m.screen = screenFollow
		m.followInput.Reset()
		m.followInput.Focus()
		m.err = ""
		return m, textinput.Blink
	case menuItemOptions:
		m.screen = screenOptions
		m.menuCur = 0
		return m, nil
	case menuItemQuit:
		return m, tea.Quit
	}
	return m, nil
}

func updateOptions(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		if key >= "1" && key <= "2" {
			idx := int(key[0] - '1')
			if idx < optItemCount {
				m.menuCur = idx
				return runOptAction(m, idx)
			}
		}
		switch key {
		case "up", "k":
			m.menuCur--
			if m.menuCur < 0 {
				m.menuCur = optItemCount - 1
			}
			return m, nil
		case "down", "j":
			m.menuCur++
			if m.menuCur >= optItemCount {
				m.menuCur = 0
			}
			return m, nil
		case "enter", " ":
			return runOptAction(m, m.menuCur)
		case "u", "b", "esc":
			m.screen = screenMenu
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func runOptAction(m model, idx int) (tea.Model, tea.Cmd) {
	switch idx {
	case optItemRelays:
		m.screen = screenRelays
		m.relayLines, m.relayURLs = buildRelayLinesAndURLs()
		m.relayCur = 0
		return m, nil
	case optItemSetKey:
		m.screen = screenSetKey
		m.setKeyInput.Reset()
		m.setKeyInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func viewMenu(m model) string {
	var lines []string
	banner := strings.Split(gostrTitle, "\n")
	lines = append(lines, banner...)
	lines = append(lines, "")
	menuStart := len(lines)
	for i, item := range menuItems {
		lines = append(lines, "  "+item.prefix+item.label)
		if i == 2 {
			lines = append(lines, "")
		}
	}
	lines = append(lines, "")
	lines = append(lines, "  "+menuItemQuitLabel)
	quitIndex := len(lines) - 1
	lines = append(lines, "")
	footerIndex := len(lines)
	lines = append(lines, "i  [1-9] select  [j/k] move  [q] quit")

	// vertical centering
	h := m.height
	if h <= 0 {
		h = 24
	}
	topPad := (h - len(lines)) / 2
	if topPad < 0 {
		topPad = 0
	}

	// horizontal centering per line
	w := m.width
	if w <= 0 {
		w = 80
	}

	var b strings.Builder
	for i := 0; i < topPad; i++ {
		b.WriteByte('\n')
	}

	for idx, line := range lines {
		// compute padding based on rune length
		runes := []rune(line)
		l := len(runes)
		pad := 0
		if l < w {
			pad = (w - l) / 2
			if pad < 0 {
				pad = 0
			}
		}
		padded := strings.Repeat(" ", pad) + line

		// decide style: menu lines get cursor highlighting (blank at menuStart+3 is not a menu item)
		menuIdx := -1
		if idx >= menuStart && idx < menuStart+len(menuItems)+1 {
			if idx < menuStart+3 {
				menuIdx = idx - menuStart
			} else if idx > menuStart+3 {
				menuIdx = idx - menuStart - 1
			}
		}
		if menuIdx >= 0 && menuIdx < len(menuItems) {
			if menuIdx == m.menuCur {
				b.WriteString(tuiStyle.Cursor.Render(padded))
			} else {
				b.WriteString(tuiStyle.Base.Render(padded))
			}
		} else if idx == quitIndex && m.menuCur == menuItemQuit {
			b.WriteString(tuiStyle.Cursor.Render(padded))
		} else if idx == quitIndex {
			b.WriteString(tuiStyle.Base.Render(padded))
		} else if idx == footerIndex {
			b.WriteString(tuiStyle.Base.Render(padded))
		} else {
			b.WriteString(tuiStyle.Base.Render(padded))
		}
		b.WriteByte('\n')
	}

	return tuiStyle.Screen.Render(b.String())
}

func viewOptions(m model) string {
	var lines []string
	lines = append(lines, "Optionen")
	lines = append(lines, "")
	optStart := len(lines)
	for _, item := range optItems {
		lines = append(lines, "  "+item.prefix+item.label)
	}
	lines = append(lines, "")
	footerIndex := len(lines)
	lines = append(lines, "i  [1-2] select  [j/k] move  [u] back  [q] quit")

	h := m.height
	if h <= 0 {
		h = 24
	}
	topPad := (h - len(lines)) / 2
	if topPad < 0 {
		topPad = 0
	}
	w := m.width
	if w <= 0 {
		w = 80
	}

	var b strings.Builder
	for i := 0; i < topPad; i++ {
		b.WriteByte('\n')
	}
	for idx, line := range lines {
		runes := []rune(line)
		l := len(runes)
		pad := 0
		if l < w {
			pad = (w - l) / 2
		}
		padded := strings.Repeat(" ", pad) + line
		if idx >= optStart && idx < optStart+len(optItems) {
			optIdx := idx - optStart
			if optIdx == m.menuCur {
				b.WriteString(tuiStyle.Cursor.Render(padded))
			} else {
				b.WriteString(tuiStyle.Base.Render(padded))
			}
		} else if idx == footerIndex {
			b.WriteString(tuiStyle.Base.Render(padded))
		} else {
			b.WriteString(tuiStyle.Base.Render(padded))
		}
		b.WriteByte('\n')
	}
	return tuiStyle.Screen.Render(b.String())
}

func buildRelayLines() []string {
	lines, _ := buildRelayLinesAndURLs()
	return lines
}

func buildRelayLinesAndURLs() (lines []string, urls []string) {
	if len(config.Relays) == 0 {
		return []string{"i  No relays configured."}, nil
	}
	lines = make([]string, 0, len(config.Relays))
	urls = make([]string, 0, len(config.Relays))
	for url, policy := range config.Relays {
		lines = append(lines, "i  "+url+"  "+policy.String())
		urls = append(urls, url)
	}
	return lines, urls
}

func buildFollowLines() []string {
	if len(config.Following) == 0 {
		return []string{"i  Not following anyone."}
	}
	lines := make([]string, 0, len(config.Following))
	for _, f := range config.Following {
		name := f.Key
		if f.Name != "" {
			name = f.Name + " (" + shorten(f.Key) + ")"
		} else {
			name = shorten(f.Key)
		}
		lines = append(lines, "1  "+name)
	}
	return lines
}

func updateRelays(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.relayCur--
			if m.relayCur < 0 {
				m.relayCur = 0
			}
			return m, nil
		case "down", "j":
			m.relayCur++
			if m.relayCur >= len(m.relayURLs) {
				m.relayCur = len(m.relayURLs) - 1
			}
			return m, nil
		case "a":
			m.screen = screenAddRelay
			m.addRelayInput.Reset()
			m.addRelayInput.Focus()
			return m, textinput.Blink
		case "r":
			if len(m.relayURLs) > 0 && m.relayCur >= 0 && m.relayCur < len(m.relayURLs) {
				removeRelayURL(m.relayURLs[m.relayCur])
				saveConfig(tuiConfigPath)
				m.relayLines, m.relayURLs = buildRelayLinesAndURLs()
				if m.relayCur >= len(m.relayURLs) {
					m.relayCur = len(m.relayURLs) - 1
				}
				if m.relayCur < 0 {
					m.relayCur = 0
				}
			}
			return m, nil
		case "u", "b", "esc":
			m.screen = screenOptions
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func viewRelays(m model) string {
	s := tuiStyle.Base.Render("i  Relays") + "\n\n"
	for i, line := range m.relayLines {
		if i == m.relayCur && len(m.relayURLs) > 0 {
			s += tuiStyle.Cursor.Render(line) + "\n"
		} else {
			s += tuiStyle.Base.Render(line) + "\n"
		}
	}
	s += "\n" + tuiStyle.Base.Render("[a] add  [r] remove  [u] back") + "\n"
	return tuiStyle.Screen.Render(s)
}

func updateFollowing(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "u", "b", "esc":
			m.screen = screenMenu
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func viewFollowing(m model) string {
	s := tuiStyle.Base.Render("i  Following") + "\n\n"
	for _, line := range m.followLines {
		s += tuiStyle.Base.Render(line) + "\n"
	}
	s += "\n" + tuiStyle.Base.Render("[u] back") + "\n"
	return tuiStyle.Screen.Render(s)
}
