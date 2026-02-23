package main

import (
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/dustin/go-humanize"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/atotto/clipboard"
)

func publishLikeCmd(targetID, authorPubkey string) tea.Cmd {
	return func() tea.Msg {
		ourID, err := PublishReaction(targetID, authorPubkey)
		return reactionDoneMsg{err: err, action: "like", targetID: targetID, ourEventID: ourID}
	}
}

func publishUnlikeCmd(ourReactionID string) tea.Cmd {
	return func() tea.Msg {
		err := PublishDeletion(ourReactionID)
		return reactionDoneMsg{err: err, action: "unlike", ourEventID: ourReactionID}
	}
}

func publishBoostCmd(ev nostr.Event) tea.Cmd {
	return func() tea.Msg {
		evJSON, _ := json.Marshal(ev)
		ourID, err := PublishBoost(ev.ID, ev.PubKey, string(evJSON))
		return reactionDoneMsg{err: err, action: "boost", targetID: ev.ID, ourEventID: ourID}
	}
}

func publishUnboostCmd(ourBoostID string) tea.Cmd {
	return func() tea.Msg {
		err := PublishDeletion(ourBoostID)
		return reactionDoneMsg{err: err, action: "unboost", ourEventID: ourBoostID}
	}
}

func updateDetail(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "r" && m.inbox && len(m.events) > 0 && m.listCur >= 0 && m.listCur < len(m.events) {
			ev := m.events[m.listCur]
			m.screen = screenComposeMessage
			m.composeFollowKeys = nil
			m.composeRecipientCur = 0
			m.composeRecipientSelected = ev.PubKey
			m.composeToInput.Reset()
			m.composeInput.Reset()
			m.composeToInput.Blur()
			authName := shorten(ev.PubKey)
			if n, ok := m.nameMap[ev.PubKey]; ok && n != "" {
				authName = n
			}
			m.composeInput.Placeholder = "Message to " + authName + "..."
			m.composeInput.Focus()
			m.err = ""
			return m, textinput.Blink
		}
	}
	switch msg := msg.(type) {
	case reactionDoneMsg:
		if msg.err != nil {
			m.detailStatus = "Error: " + msg.err.Error()
			return m, nil
		}
		switch msg.action {
		case "like":
			if msg.targetID != "" && msg.ourEventID != "" {
				if m.likedMap == nil {
					m.likedMap = make(map[string]string)
				}
				m.likedMap[msg.targetID] = msg.ourEventID
			}
			m.detailStatus = ""
		case "unlike":
			for tid, oid := range m.likedMap {
				if oid == msg.ourEventID {
					delete(m.likedMap, tid)
					break
				}
			}
			m.detailStatus = ""
		case "boost":
			if msg.targetID != "" && msg.ourEventID != "" {
				if m.boostedMap == nil {
					m.boostedMap = make(map[string]string)
				}
				m.boostedMap[msg.targetID] = msg.ourEventID
			}
			m.detailStatus = ""
		case "unboost":
			for tid, oid := range m.boostedMap {
				if oid == msg.ourEventID {
					delete(m.boostedMap, tid)
					break
				}
			}
			m.detailStatus = ""
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "u", "esc", "q":
			m.screen = screenList
			m.detailStatus = ""
			return m, nil
		case "c":
			if len(m.events) > 0 && m.listCur >= 0 && m.listCur < len(m.events) {
				ev := m.events[m.listCur]
				npub, err := nip19.EncodePublicKey(ev.PubKey, "")
				if err == nil {
					_ = clipboard.WriteAll(npub)
					m.detailStatus = "npub copied"
				}
			}
			return m, nil
		case "b":
			if len(m.events) == 0 || m.listCur < 0 || m.listCur >= len(m.events) {
				return m, nil
			}
			if config.PrivateKey == "" {
				m.detailStatus = "Set key first (Optionen)"
				return m, nil
			}
			ev := m.events[m.listCur]
			if ev.Kind != nostr.KindTextNote {
				return m, nil
			}
			if ourID, ok := m.boostedMap[ev.ID]; ok {
				return m, publishUnboostCmd(ourID)
			}
			return m, publishBoostCmd(ev)
		case "l":
			if len(m.events) == 0 || m.listCur < 0 || m.listCur >= len(m.events) {
				return m, nil
			}
			if config.PrivateKey == "" {
				m.detailStatus = "Set key first (Optionen)"
				return m, nil
			}
			ev := m.events[m.listCur]
			if ourID, ok := m.likedMap[ev.ID]; ok {
				return m, publishUnlikeCmd(ourID)
			}
			return m, publishLikeCmd(ev.ID, ev.PubKey)
		default:
			m.detailStatus = ""
			return m, nil
		}
	}
	return m, nil
}

func viewDetail(m model) string {
	if len(m.events) == 0 || m.listCur < 0 || m.listCur >= len(m.events) {
		return tuiStyle.Screen.Render(tuiStyle.Base.Render("i  No event selected."))
	}
	ev := m.events[m.listCur]
	author := shorten(ev.PubKey)
	if n, ok := m.nameMap[ev.PubKey]; ok && n != "" {
		author = n
	}
	width := m.width - 4
	if width < 40 {
		width = 40
	}
	s := tuiStyle.Base.Render("0  "+ev.ID) + "\n\n"
	s += tuiStyle.Base.Render("from "+author+"  "+humanize.Time(ev.CreatedAt)) + "\n\n"
	content := ev.Content
	wrapped := wrap(content, width)
	for _, line := range strings.Split(wrapped, "\n") {
		s += tuiStyle.Base.Render("  "+line) + "\n"
	}
	if m.detailStatus != "" {
		s += "\n" + tuiStyle.Base.Render("i  "+m.detailStatus) + "\n"
	}
	liked := m.likedMap != nil && m.likedMap[ev.ID] != ""
	boosted := m.boostedMap != nil && m.boostedMap[ev.ID] != ""
	likeStr := "[l] like "
	if liked {
		likeStr += "\u2713"
	}
	boostStr := "[b] boost "
	if boosted {
		boostStr += "\u2713"
	}
	footer := likeStr + "  " + boostStr + "  [c] copy npub"
	if m.inbox {
		footer += "  [r] reply"
	}
	footer += "  [u] back"
	s += "\n" + tuiStyle.Base.Render(footer) + "\n"
	return tuiStyle.Screen.Render(s)
}
