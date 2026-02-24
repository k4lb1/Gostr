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

func detailCurrentEvent(m model) *nostr.Event {
	if len(m.detailStack) == 0 {
		return nil
	}
	ev := m.detailStack[len(m.detailStack)-1]
	return &ev
}

func updateDetail(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	// repliesLoadedMsg must be handled first (can arrive while in detail)
	switch msg := msg.(type) {
	case repliesLoadedMsg:
		if len(m.detailStack) == 0 || m.detailStack[len(m.detailStack)-1].ID != msg.rootID {
			return m, nil
		}
		m.detailReplies = msg.replies
		m.detailRepliesLoading = false
		m.detailReplyCur = 0
		for k, v := range msg.nameMap {
			if v != "" {
				m.nameMap[k] = v
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "r" && m.inbox && len(m.detailStack) > 0 {
			ev := m.detailStack[len(m.detailStack)-1]
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
		if msg.String() == "r" && !m.inbox && len(m.detailStack) > 0 && config.PrivateKey != "" {
			ev := m.detailStack[len(m.detailStack)-1]
			if ev.Kind == nostr.KindTextNote {
				m.screen = screenComposeNote
				m.composeReplyTargetID = ev.ID
				m.composeReplyTargetAuthor = ev.PubKey
				m.composeReplyRootID = m.detailStack[0].ID
				m.composeReplyRootAuthor = m.detailStack[0].PubKey
				m.composeInput.Reset()
				authName := shorten(ev.PubKey)
				if n, ok := m.nameMap[ev.PubKey]; ok && n != "" {
					authName = n
				}
				m.composeInput.Placeholder = "Reply to " + authName + "..."
				m.composeInput.Focus()
				m.err = ""
				return m, textinput.Blink
			}
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
		ev := detailCurrentEvent(m)
		switch msg.String() {
		case "u", "esc", "q":
			if len(m.detailStack) <= 1 {
				m.screen = screenList
				m.detailStack = nil
				m.detailReplies = nil
				m.detailStatus = ""
				return m, nil
			}
			m.detailStack = m.detailStack[:len(m.detailStack)-1]
			m.detailReplies = nil
			m.detailRepliesLoading = true
			m.detailReplyCur = 0
			m.detailStatus = ""
			return m, loadRepliesCmd(m.detailStack[len(m.detailStack)-1].ID)
		case "down", "j":
			if len(m.detailReplies) > 0 && m.detailReplyCur < len(m.detailReplies)-1 {
				m.detailReplyCur++
				m.detailStatus = ""
			}
			return m, nil
		case "up", "k":
			if m.detailReplyCur > 0 {
				m.detailReplyCur--
				m.detailStatus = ""
			}
			return m, nil
		case "enter", " ":
			if len(m.detailReplies) > 0 && m.detailReplyCur >= 0 && m.detailReplyCur < len(m.detailReplies) {
				reply := m.detailReplies[m.detailReplyCur]
				m.detailStack = append(m.detailStack, reply)
				m.detailReplies = nil
				m.detailRepliesLoading = true
				m.detailReplyCur = 0
				m.detailStatus = ""
				return m, loadRepliesCmd(reply.ID)
			}
			return m, nil
		case "c":
			if ev != nil {
				npub, err := nip19.EncodePublicKey(ev.PubKey, "")
				if err == nil {
					_ = clipboard.WriteAll(npub)
					m.detailStatus = "npub copied"
				}
			}
			return m, nil
		case "b":
			if ev == nil || config.PrivateKey == "" {
				if config.PrivateKey == "" && ev != nil {
					m.detailStatus = "Set key first (Optionen)"
				}
				return m, nil
			}
			if ev.Kind != nostr.KindTextNote {
				return m, nil
			}
			if ourID, ok := m.boostedMap[ev.ID]; ok {
				return m, publishUnboostCmd(ourID)
			}
			return m, publishBoostCmd(*ev)
		case "l":
			if ev == nil || config.PrivateKey == "" {
				if config.PrivateKey == "" && ev != nil {
					m.detailStatus = "Set key first (Optionen)"
				}
				return m, nil
			}
			if ourID, ok := m.likedMap[ev.ID]; ok {
				return m, publishUnlikeCmd(ourID)
			}
			return m, publishLikeCmd(ev.ID, ev.PubKey)
		case "i":
			if ev == nil || !config.AllowImageASCII {
				return m, nil
			}
			urls := extractImageURLs(ev.Content)
			if len(urls) == 0 {
				return m, nil
			}
			if len(urls) == 1 {
				m.screen = screenImageASCII
				m.imageASCIIContent = ""
				m.imageASCIILoading = true
				return m, loadImageASCIICmd(urls[0])
			}
			m.screen = screenImageURLSelect
			m.imageURLs = urls
			m.imageURLCur = 0
			return m, nil
		default:
			m.detailStatus = ""
			return m, nil
		}
	}
	return m, nil
}

func viewDetail(m model) string {
	if len(m.detailStack) == 0 {
		return tuiStyle.Screen.Render(tuiStyle.Base.Render("i  No event selected."))
	}
	ev := m.detailStack[len(m.detailStack)-1]
	author := shorten(ev.PubKey)
	if n, ok := m.nameMap[ev.PubKey]; ok && n != "" {
		author = n
	}
	width := m.width - 4
	if width < 40 {
		width = 40
	}
	s := tuiStyle.Base.Render("0  "+ev.ID) + "\n\n"
	s += tuiStyle.Base.Render("i  from "+author+"  "+humanize.Time(ev.CreatedAt)) + "\n\n"
	content := ev.Content
	wrapped := wrap(content, width)
	for _, line := range strings.Split(wrapped, "\n") {
		s += tuiStyle.Base.Render("  "+line) + "\n"
	}

	// Replies section
	if m.detailRepliesLoading {
		s += "\n" + tuiStyle.Base.Render("i  Loading replies...") + "\n"
	} else if len(m.detailReplies) > 0 {
		s += "\n" + tuiStyle.Base.Render("1  Replies") + "\n\n"
		for i, reply := range m.detailReplies {
			replyAuthor := shorten(reply.PubKey)
			if n, ok := m.nameMap[reply.PubKey]; ok && n != "" {
				replyAuthor = n
			}
			preview := strings.ReplaceAll(reply.Content, "\n", " ")
			preview = strings.TrimSpace(preview)
			if len([]rune(preview)) > width-12 {
				preview = string([]rune(preview)[:width-15]) + "..."
			}
			line := "0  [" + replyAuthor + "] " + preview
			if i == m.detailReplyCur {
				s += tuiStyle.Cursor.Render(line) + "\n"
			} else {
				s += tuiStyle.Base.Render(line) + "\n"
			}
		}
		s += "\n"
	}

	if m.detailStatus != "" {
		s += tuiStyle.Base.Render("i  "+m.detailStatus) + "\n"
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
	} else if ev.Kind == nostr.KindTextNote {
		footer += "  [r] reply"
	}
	if config.AllowImageASCII && len(extractImageURLs(ev.Content)) > 0 {
		footer += "  [i] image as ASCII"
	}
	footer += "  [u] back"
	if len(m.detailReplies) > 0 {
		footer += "  [j/k] replies  [enter] open"
	}
	s += "\n" + tuiStyle.Base.Render("i  "+footer) + "\n"
	return tuiStyle.Screen.Render(s)
}

func updateImageURLSelect(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "u", "esc", "q":
			m.screen = screenDetail
			m.imageURLs = nil
			m.imageURLCur = 0
			return m, nil
		case "down", "j":
			m.imageURLCur++
			if m.imageURLCur >= len(m.imageURLs) {
				m.imageURLCur = len(m.imageURLs) - 1
			}
			return m, nil
		case "up", "k":
			m.imageURLCur--
			if m.imageURLCur < 0 {
				m.imageURLCur = 0
			}
			return m, nil
		case "enter", " ":
			if len(m.imageURLs) > 0 && m.imageURLCur >= 0 && m.imageURLCur < len(m.imageURLs) {
				m.screen = screenImageASCII
				m.imageASCIIContent = ""
				m.imageASCIILoading = true
				return m, loadImageASCIICmd(m.imageURLs[m.imageURLCur])
			}
			return m, nil
		}
	}
	return m, nil
}

func viewImageURLSelect(m model) string {
	s := tuiStyle.Base.Render("i  Select image URL") + "\n\n"
	for i, url := range m.imageURLs {
		short := url
		if len([]rune(url)) > m.width-6 {
			short = string([]rune(url)[:m.width-9]) + "..."
		}
		line := "0  " + short
		if i == m.imageURLCur {
			s += tuiStyle.Cursor.Render(line) + "\n"
		} else {
			s += tuiStyle.Base.Render(line) + "\n"
		}
	}
	s += "\n" + tuiStyle.Base.Render("i  [j/k] select  [enter] load  [u] back") + "\n"
	return tuiStyle.Screen.Render(s)
}

func updateImageASCII(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case imageASCIILoadedMsg:
		m.imageASCIILoading = false
		if msg.err != nil {
			m.imageASCIIContent = "i  Error: " + msg.err.Error()
		} else {
			m.imageASCIIContent = msg.content
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "u", "esc", "q":
			m.screen = screenDetail
			m.imageASCIIContent = ""
			m.imageASCIILoading = false
			return m, nil
		}
	}
	return m, nil
}

func viewImageASCII(m model) string {
	s := tuiStyle.Base.Render("i  Image as ASCII") + "\n\n"
	if m.imageASCIILoading {
		s += tuiStyle.Base.Render("i  Loading...") + "\n"
	} else if m.imageASCIIContent != "" {
		s += m.imageASCIIContent + "\n"
	}
	s += "\n" + tuiStyle.Base.Render("i  [u] back") + "\n"
	return tuiStyle.Screen.Render(s)
}
