package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nbd-wtf/go-nostr"
)

func updateList(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case homeLoadedMsg:
		m.events = msg.events
		m.nameMap = msg.nameMap
		m.likedMap = msg.likedMap
		m.boostedMap = msg.boostedMap
		m.inbox = msg.inbox
		m.aether = msg.aether
		m.loading = false
		m.listCur = 0
		m.listOffset = 0
		if msg.errMsg != "" {
			m.err = msg.errMsg
		} else if len(m.events) == 0 {
			m.err = "No events"
		} else {
			m.err = ""
		}
		return m, nil
	case tea.KeyMsg:
		if m.loading {
			// allow backing out while loading, but ignore other keys
			switch msg.String() {
			case "u", "b", "esc":
				m.screen = screenMenu
				m.events = nil
				m.err = ""
				m.loading = false
				return m, nil
			}
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			m.listCur--
			if m.listCur < 0 {
				m.listCur = 0
			}
			m.listOffset = clampListOffset(m)
			return m, nil
		case "down", "j":
			m.listCur++
			if m.listCur >= len(m.events) {
				m.listCur = len(m.events) - 1
			}
			m.listOffset = clampListOffset(m)
			return m, nil
		case "enter", " ":
			if len(m.events) > 0 && m.listCur >= 0 && m.listCur < len(m.events) {
				m.screen = screenDetail
			}
			return m, nil
		case "r":
			m.loading = true
			m.events = nil
			return m, loadFeedCmd(m.inbox, m.notesOnly, m.aether)
		case "tab":
			m.loading = true
			m.events = nil
			m.inbox = !m.inbox
			m.aether = false
			return m, loadFeedCmd(m.inbox, m.notesOnly, m.aether)
		case "m":
			if m.inbox {
				m.screen = screenComposeMessage
				m.composeFollowKeys = buildComposeFollowKeys()
				m.composeRecipientCur = 0
				m.composeRecipientSelected = ""
				m.composeToInput.Reset()
				m.composeInput.Reset()
				m.composeToInput.Blur()
				m.composeInput.Blur()
				m.err = ""
				return m, nil
			}
			return m, nil
		case "u", "b", "esc":
			m.screen = screenMenu
			m.events = nil
			m.err = ""
			return m, nil
		}
	}
	return m, nil
}

func clampListOffset(m model) int {
	linesPerItem := 3
	contentLines := m.height - 4
	if contentLines < 6 {
		contentLines = 6
	}
	visibleItems := contentLines / linesPerItem
	if visibleItems < 1 {
		visibleItems = 1
	}
	n := len(m.events)
	if n == 0 {
		return 0
	}
	offset := m.listOffset
	if m.listCur < offset {
		offset = m.listCur
	}
	if m.listCur >= offset+visibleItems {
		offset = m.listCur - visibleItems + 1
	}
	if offset < 0 {
		offset = 0
	}
	if offset+visibleItems > n {
		offset = n - visibleItems
		if offset < 0 {
			offset = 0
		}
	}
	return offset
}

func viewList(m model) string {
	title := "0  Home"
	if m.inbox {
		title = "5  Inbox"
	} else if m.aether {
		title = "3  Aether"
	} else if !m.notesOnly {
		title = "1  Notes + Comments"
	}
	s := tuiStyle.Base.Render(title) + "\n\n"
	if m.loading {
		s += tuiStyle.Base.Render("i  Loading...") + "\n"
		return tuiStyle.Screen.Render(s)
	}
	if m.err != "" {
		s += tuiStyle.Base.Render("i  "+m.err) + "\n"
		footer := "i  [u] back to menu"
		if m.inbox {
			footer += "  [m] new message"
		}
		s += tuiStyle.Base.Render(footer) + "\n"
		return tuiStyle.Screen.Render(s)
	}
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}
	// viewport: only render items listOffset..listOffset+visibleCount
	linesPerItem := 3 // 2 content lines + 1 separator
	contentLines := m.height - 4
	if contentLines < 6 {
		contentLines = 6
	}
	visibleItems := contentLines / linesPerItem
	if visibleItems < 1 {
		visibleItems = 1
	}
	start := m.listOffset
	end := start + visibleItems
	if end > len(m.events) {
		end = len(m.events)
	}
	for i := start; i < end; i++ {
		ev := m.events[i]
		lines := listLinesForEvent(ev, m.nameMap, contentWidth, i+1)
		for _, line := range lines {
			if i == m.listCur {
				s += tuiStyle.Cursor.Render(line) + "\n"
			} else {
				s += tuiStyle.Base.Render(line) + "\n"
			}
		}
		if i < len(m.events)-1 {
			s += tuiStyle.Base.Render("  " + strings.Repeat("\u2500", 24) + "  ") + "\n"
		}
	}
	footer := "i  [j/k] nav  [enter] open  [r] refresh  [tab] feed  [u] back"
	if m.inbox {
		footer += "  [m] new message"
	}
	s += tuiStyle.Base.Render(footer) + "\n"
	return tuiStyle.Screen.Render(s)
}

// listLinesForEvent returns 1–2 lines for the list: number, author, content preview.
func listLinesForEvent(ev nostr.Event, nameMap map[string]string, contentWidth int, num int) []string {
	author := shorten(ev.PubKey)
	if n, ok := nameMap[ev.PubKey]; ok && n != "" {
		author = n
	}
	content := strings.ReplaceAll(ev.Content, "\n", " ")
	content = strings.ReplaceAll(content, "\t", " ")
	content = strings.TrimSpace(content)
	var numStr string
	if num < 10 {
		numStr = " " + string(rune('0'+num))
	} else if num < 100 {
		numStr = string(rune('0'+num/10)) + string(rune('0'+num%10))
	} else {
		numStr = ".."
	}
	prefix := "  " + numStr + "  [" + author + "] "
	available := contentWidth - len([]rune(prefix))
	if available < 20 {
		available = 20
	}
	wrapped := wrap(content, available)
	lines := strings.Split(wrapped, "\n")
	out := make([]string, 0, 2)
	out = append(out, prefix+strings.TrimSpace(lines[0]))
	if len(lines) > 1 {
		out = append(out, "       "+strings.TrimSpace(lines[1]))
	}
	truncated := len(lines) > 2 || (len(lines) == 1 && len([]rune(content)) > available) || (len(lines) == 2 && len([]rune(content)) > 2*available)
	if truncated {
		last := len(out) - 1
		out[last] = out[last] + "..."
	}
	return out
}
