package main

import (
	"encoding/hex"
	"log"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip04"
)

func translatePubkey(raw string) string {
	return nip19.TranslatePublicKey(raw)
}

func buildComposeFollowKeys() []string {
	keys := make([]string, 0, len(config.Following))
	for k := range config.Following {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func updateFollow(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.screen = screenMenu
			m.followInput.Blur()
			m.err = ""
			return m, nil
		case "enter":
			raw := m.followInput.Value()
			if raw == "" {
				return m, nil
			}
			key := translatePubkey(raw)
			if key == "" {
				m.err = "Invalid pubkey (use npub or hex)"
				return m, nil
			}
			config.Following[key] = Follow{Key: key}
			saveConfig(tuiConfigPath)
			m.screen = screenMenu
			m.followInput.Blur()
			m.followInput.Reset()
			m.err = ""
			m.followLines = buildFollowLines()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.followInput, cmd = m.followInput.Update(msg)
	return m, cmd
}

func viewFollow(m model) string {
	s := tuiStyle.Base.Render("5  Follow") + "\n\n"
	if m.err != "" {
		s += tuiStyle.Base.Render("i  "+m.err) + "\n"
	}
	s += tuiStyle.Base.Render("i  Enter pubkey (npub or hex) to follow. Esc to cancel.") + "\n\n"
	s += tuiStyle.Base.Render("Pubkey: ") + m.followInput.View() + "\n"
	return tuiStyle.Screen.Render(s)
}

func updateComposeNote(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.composeReplyTargetID != "" {
				m.screen = screenDetail
				m.composeReplyTargetID = ""
				m.composeReplyTargetAuthor = ""
				m.composeReplyRootID = ""
				m.composeReplyRootAuthor = ""
			} else {
				m.screen = screenMenu
			}
			m.composeInput.Blur()
			m.err = ""
			return m, nil
		case "enter":
			content := m.composeInput.Value()
			if content == "" {
				return m, nil
			}
			if config.PrivateKey == "" {
				m.err = "Set key first"
				return m, nil
			}
			var err error
			if m.composeReplyTargetID != "" {
				err = PublishReply(m.composeReplyRootID, m.composeReplyRootAuthor, m.composeReplyTargetID, m.composeReplyTargetAuthor, content)
			} else {
				err = publishNote(content)
			}
			m.composeInput.Blur()
			m.composeInput.Reset()
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			if m.composeReplyTargetID != "" {
				m.screen = screenDetail
				targetID := m.composeReplyTargetID
				m.composeReplyTargetID = ""
				m.composeReplyTargetAuthor = ""
				m.composeReplyRootID = ""
				m.composeReplyRootAuthor = ""
				m.detailRepliesLoading = true
				m.err = ""
				return m, loadRepliesCmd(targetID)
			}
			m.screen = screenMenu
			m.err = ""
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.composeInput, cmd = m.composeInput.Update(msg)
	return m, cmd
}

func viewComposeNote(m model) string {
	title := "3  Publish note"
	if m.composeReplyTargetID != "" {
		title = "0  Reply"
	}
	s := tuiStyle.Base.Render(title) + "\n\n"
	if m.err != "" {
		s += tuiStyle.Base.Render("i  "+m.err) + "\n"
	}
	s += tuiStyle.Base.Render("i  Enter content. Enter to publish, Esc to cancel.") + "\n\n"
	s += m.composeInput.View() + "\n"
	return tuiStyle.Screen.Render(s)
}

func publishNote(content string) error {
	if config.PrivateKey == "" {
		return nil
	}
	initNostr()
	ev := nostr.Event{
		CreatedAt: time.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nil,
		Content:   content,
	}
	_, _, err := pool.PublishEvent(&ev)
	return err
}

func updateComposeMessage(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.screen = screenList
			m.composeToInput.Blur()
			m.composeInput.Blur()
			m.composeRecipientSelected = ""
			m.err = ""
			return m, nil
		case "up", "k":
			if m.composeRecipientSelected == "" && !m.composeToInput.Focused() {
				m.composeRecipientCur--
				if m.composeRecipientCur < 0 {
					m.composeRecipientCur = 0
				}
				return m, nil
			}
		case "down", "j":
			if m.composeRecipientSelected == "" && !m.composeToInput.Focused() {
				maxCur := len(m.composeFollowKeys)
				m.composeRecipientCur++
				if m.composeRecipientCur > maxCur {
					m.composeRecipientCur = maxCur
				}
				return m, nil
			}
		case "enter":
			if m.composeRecipientSelected == "" && !m.composeToInput.Focused() {
				if m.composeRecipientCur < len(m.composeFollowKeys) {
					key := m.composeFollowKeys[m.composeRecipientCur]
					m.composeRecipientSelected = key
					f := config.Following[key]
					if f.Name != "" {
						m.composeInput.Placeholder = "Message to " + f.Name + "..."
					} else {
						m.composeInput.Placeholder = "Message to " + shorten(key) + "..."
					}
					m.composeInput.Focus()
					return m, textinput.Blink
				}
				m.composeToInput.Focus()
				return m, textinput.Blink
			}
			if m.composeToInput.Focused() {
				to := m.composeToInput.Value()
				if to == "" {
					return m, nil
				}
				key := translatePubkey(to)
				if key == "" {
					m.err = "Invalid pubkey"
					return m, nil
				}
				m.composeToInput.Blur()
				m.composeRecipientSelected = key
				m.composeInput.Placeholder = "Message to " + shorten(key) + "..."
				m.composeInput.Focus()
				return m, textinput.Blink
			}
			content := m.composeInput.Value()
			if content == "" {
				return m, nil
			}
			if config.PrivateKey == "" {
				m.err = "Set key first"
				return m, nil
			}
			toKey := m.composeRecipientSelected
			if toKey == "" {
				toKey = translatePubkey(m.composeToInput.Value())
			}
			if toKey == "" {
				m.err = "Invalid recipient"
				return m, nil
			}
			err := sendEncryptedDM(toKey, content)
			m.composeInput.Blur()
			m.composeToInput.Blur()
			m.composeInput.Reset()
			m.composeToInput.Reset()
			m.composeRecipientSelected = ""
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.screen = screenList
			m.err = ""
			return m, nil
		}
	}
	var cmd tea.Cmd
	if m.composeToInput.Focused() {
		m.composeToInput, cmd = m.composeToInput.Update(msg)
	} else {
		m.composeInput, cmd = m.composeInput.Update(msg)
	}
	return m, cmd
}

func viewComposeMessage(m model) string {
	s := tuiStyle.Base.Render("New message") + "\n\n"
	if m.err != "" {
		s += tuiStyle.Base.Render("i  "+m.err) + "\n"
	}
	if m.composeRecipientSelected != "" {
		recipLabel := shorten(m.composeRecipientSelected)
		if f, ok := config.Following[m.composeRecipientSelected]; ok && f.Name != "" {
			recipLabel = f.Name + " (" + shorten(m.composeRecipientSelected) + ")"
		}
		s += tuiStyle.Base.Render("To: "+recipLabel) + "\n\n"
		s += tuiStyle.Base.Render("Message: ") + m.composeInput.View() + "\n"
		s += tuiStyle.Base.Render("i  Enter message, then Enter to send.") + "\n"
	} else if m.composeToInput.Focused() {
		s += tuiStyle.Base.Render("To: ") + m.composeToInput.View() + "\n"
		s += tuiStyle.Base.Render("i  Enter recipient pubkey, then Enter.") + "\n"
	} else {
		s += tuiStyle.Base.Render("i  Select recipient (j/k) or enter npub manually:") + "\n\n"
		for i, key := range m.composeFollowKeys {
			f := config.Following[key]
			name := shorten(key)
			if f.Name != "" {
				name = f.Name + " (" + shorten(key) + ")"
			}
			line := "  " + name
			if i == m.composeRecipientCur {
				s += tuiStyle.Cursor.Render(line) + "\n"
			} else {
				s += tuiStyle.Base.Render(line) + "\n"
			}
		}
		manualLine := "  Enter npub manually..."
		if m.composeRecipientCur == len(m.composeFollowKeys) {
			s += tuiStyle.Cursor.Render(manualLine) + "\n"
		} else {
			s += tuiStyle.Base.Render(manualLine) + "\n"
		}
		s += "\n" + tuiStyle.Base.Render("i  [j/k] move  [enter] select  [esc] back") + "\n"
	}
	return tuiStyle.Screen.Render(s)
}

func sendEncryptedDM(toPubkey, content string) error {
	initNostr()
	skHex := hex.EncodeToString([]byte(config.PrivateKey))
	sharedSecret, err := nip04.ComputeSharedSecret(skHex, toPubkey)
	if err != nil {
		return err
	}
	encrypted, err := nip04.Encrypt(content, sharedSecret)
	if err != nil {
		return err
	}
	ev := nostr.Event{
		CreatedAt: time.Now(),
		Kind:      nostr.KindEncryptedDirectMessage,
		Tags:      nostr.Tags{{"p", toPubkey}},
		Content:   encrypted,
	}
	_, _, err = pool.PublishEvent(&ev)
	if err != nil {
		log.Printf("send DM: %v", err)
		return err
	}
	return nil
}
