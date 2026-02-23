package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

func updateSetKey(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.screen = screenOptions
			return m, nil
		case "enter":
			val := m.setKeyInput.Value()
			if val == "" {
				return m, nil
			}
			if err := setPrivateKeyString(val); err != nil {
				m.err = "Invalid key: " + err.Error()
				return m, nil
			}
			saveConfig(tuiConfigPath)
			m.screen = screenOptions
			m.err = ""
			m.setKeyInput.Blur()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.setKeyInput, cmd = m.setKeyInput.Update(msg)
	return m, cmd
}

func viewSetKey(m model) string {
	s := tuiStyle.Base.Render("2  Set private key (nsec or hex)") + "\n\n"
	if m.err != "" {
		s += tuiStyle.Base.Render("i  "+m.err) + "\n"
	}
	s += tuiStyle.Base.Render("i  Paste key and press Enter. Esc to cancel.") + "\n\n"
	s += m.setKeyInput.View() + "\n"
	return tuiStyle.Screen.Render(s)
}

func updateAddRelay(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.screen = screenRelays
			m.addRelayInput.Blur()
			m.relayLines, m.relayURLs = buildRelayLinesAndURLs()
			return m, nil
		case "enter":
			url := m.addRelayInput.Value()
			if url == "" {
				return m, nil
			}
			addRelayURL(url)
			saveConfig(tuiConfigPath)
			m.screen = screenRelays
			m.addRelayInput.Blur()
			m.relayLines, m.relayURLs = buildRelayLinesAndURLs()
			m.relayCur = 0
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.addRelayInput, cmd = m.addRelayInput.Update(msg)
	return m, cmd
}

func viewAddRelay(m model) string {
	s := tuiStyle.Base.Render("i  Add relay") + "\n\n"
	s += tuiStyle.Base.Render("i  Enter relay URL (e.g. wss://relay.damus.io). Esc to cancel.") + "\n\n"
	s += m.addRelayInput.View() + "\n"
	return tuiStyle.Screen.Render(s)
}
