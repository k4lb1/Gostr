package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/nbd-wtf/go-nostr"
)

type logFilterWriter struct {
	w io.Writer
}

func (f *logFilterWriter) Write(p []byte) (n int, err error) {
	if strings.Contains(strings.ToLower(string(p)), "bad signature") {
		return len(p), nil
	}
	return f.w.Write(p)
}

var tuiConfigPath string

// installLogFilter sets a log writer that filters "bad signature" messages.
func installLogFilter() {
	log.SetOutput(&logFilterWriter{w: os.Stderr})
}

type screen int

const (
	screenMenu screen = iota
	screenOptions
	screenList
	screenDetail
	screenRelays
	screenFollowing
	screenSetKey
	screenAddRelay
	screenFollow
	screenComposeNote
	screenComposeMessage
)

const feedLimit = 25

type model struct {
	screen       screen
	width        int
	height       int
	menuCur      int
	listCur      int
	listOffset   int // first visible item index for scrolling
	relayCur     int
	events       []nostr.Event
	nameMap      map[string]string
	likedMap     map[string]string   // target ev ID -> our reaction ev ID
	boostedMap   map[string]string   // target ev ID -> our boost ev ID
	loading      bool
	inbox        bool
	notesOnly    bool // when true (Home): show only top-level notes, no replies
	aether       bool // when true: unfiltered notes from all
	err          string
	relayLines   []string
	relayURLs    []string
	followLines  []string
	setKeyInput     textinput.Model
	addRelayInput   textinput.Model
	followInput     textinput.Model
	composeInput       textinput.Model
	composeToInput     textinput.Model
	composeFollowKeys  []string // sorted pubkeys from Following, for recipient selection
	composeRecipientCur int
	composeRecipientSelected string // hex pubkey when chosen from list
	detailStatus       string
}


type backMsg struct{}
type homeLoadedMsg struct {
	events     []nostr.Event
	nameMap    map[string]string
	likedMap   map[string]string
	boostedMap map[string]string
	inbox      bool
	aether     bool
	errMsg     string
}
type relayListMsg struct{ lines []string }
type followListMsg struct{ lines []string }
type reactionDoneMsg struct {
	err        error
	action     string // "like" "unlike" "boost" "unboost"
	targetID   string
	ourEventID string
}

func runTUI(configPath string) {
	tuiConfigPath = configPath
	prog := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		panic(err)
	}
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "nsec or hex..."
	ti.Width = 50
	ti.PromptStyle = tuiStyle.Base
	ti.TextStyle = tuiStyle.Base
	ar := textinput.New()
	ar.Placeholder = "wss://..."
	ar.Width = 50
	ar.PromptStyle = tuiStyle.Base
	ar.TextStyle = tuiStyle.Base
	fi := textinput.New()
	fi.Placeholder = "npub or hex..."
	fi.Width = 60
	fi.PromptStyle = tuiStyle.Base
	fi.TextStyle = tuiStyle.Base
	ci := textinput.New()
	ci.Placeholder = "Your note..."
	ci.Width = 60
	ci.PromptStyle = tuiStyle.Base
	ci.TextStyle = tuiStyle.Base
	cti := textinput.New()
	cti.Placeholder = "npub or hex..."
	cti.Width = 60
	cti.PromptStyle = tuiStyle.Base
	cti.TextStyle = tuiStyle.Base
	return model{
		screen:        screenMenu,
		width:         80,
		height:        24,
		menuCur:       0,
		listCur:       0,
		relayCur:      0,
		events:        nil,
		nameMap:       make(map[string]string),
		likedMap:      make(map[string]string),
		boostedMap:    make(map[string]string),
		loading:       false,
		inbox:         false,
		notesOnly:     true,
		relayLines:    nil,
		relayURLs:     nil,
		followLines:   nil,
		setKeyInput:   ti,
		addRelayInput: ar,
		followInput:   fi,
		composeInput:  ci,
		composeToInput: cti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width < 40 {
			m.width = 40
		}
		if m.height < 10 {
			m.height = 10
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.screen == screenMenu || m.screen == screenOptions {
				return m, tea.Quit
			}
		}
	}

	switch m.screen {
	case screenMenu:
		return updateMenu(m, msg)
	case screenOptions:
		return updateOptions(m, msg)
	case screenList:
		return updateList(m, msg)
	case screenDetail:
		return updateDetail(m, msg)
	case screenRelays:
		return updateRelays(m, msg)
	case screenFollowing:
		return updateFollowing(m, msg)
	case screenSetKey:
		return updateSetKey(m, msg)
	case screenAddRelay:
		return updateAddRelay(m, msg)
	case screenFollow:
		return updateFollow(m, msg)
	case screenComposeNote:
		return updateComposeNote(m, msg)
	case screenComposeMessage:
		return updateComposeMessage(m, msg)
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenMenu:
		return viewMenu(m)
	case screenOptions:
		return viewOptions(m)
	case screenList:
		return viewList(m)
	case screenDetail:
		return viewDetail(m)
	case screenRelays:
		return viewRelays(m)
	case screenFollowing:
		return viewFollowing(m)
	case screenSetKey:
		return viewSetKey(m)
	case screenAddRelay:
		return viewAddRelay(m)
	case screenFollow:
		return viewFollow(m)
	case screenComposeNote:
		return viewComposeNote(m)
	case screenComposeMessage:
		return viewComposeMessage(m)
	}
	return ""
}

// loadHomeFeed runs in background and sends homeLoadedMsg
func loadHomeFeed(inbox, notesOnly, aether bool) tea.Msg {
	var keys []string
	nameMap := make(map[string]string)
	if !aether {
		for _, follow := range config.Following {
			keys = append(keys, follow.Key)
			if follow.Name != "" {
				nameMap[follow.Key] = follow.Name
			}
		}
	}
	if !inbox && !aether && len(keys) == 0 {
		return homeLoadedMsg{events: nil, nameMap: nameMap, likedMap: make(map[string]string), boostedMap: make(map[string]string), inbox: inbox, aether: false, errMsg: "Follow someone first"}
	}
	if inbox && config.PrivateKey == "" {
		return homeLoadedMsg{events: nil, nameMap: nameMap, likedMap: make(map[string]string), boostedMap: make(map[string]string), inbox: inbox, aether: false, errMsg: "Set key first"}
	}
	initNostr()
	var pubkey string
	if config.PrivateKey != "" {
		pubkey = getPubKey(config.PrivateKey)
	}
	filters := nostr.Filters{{Limit: feedLimit}}
	if inbox {
		filters[0].Tags = nostr.TagMap{"p": {pubkey}}
		filters[0].Kinds = []int{nostr.KindEncryptedDirectMessage}
	} else if aether {
		filters[0].Kinds = []int{nostr.KindTextNote}
	} else {
		filters[0].Authors = keys
		filters[0].Kinds = []int{nostr.KindTextNote}
	}
	_, all := pool.Sub(filters)
	var events []nostr.Event
	ch := iterEventsWithTimeout(nostr.Unique(all), 2*time.Second)
	for ev := range ch {
		if inbox {
			events = append(events, ev)
		} else if aether {
			events = append(events, ev)
		} else if notesOnly {
			hasE := false
			for _, tag := range ev.Tags {
				if len(tag) > 0 && tag[0] == "e" {
					hasE = true
					break
				}
			}
			if !hasE {
				events = append(events, ev)
			}
		} else {
			events = append(events, ev)
		}
		if len(events) >= feedLimit {
			break
		}
	}
	// fetch Kind 0 metadata for authors we don't have names for
	nameMap = fillNameMap(events, nameMap)
	likedMap, boostedMap := loadOurReactions(events)
	return homeLoadedMsg{events: events, nameMap: nameMap, likedMap: likedMap, boostedMap: boostedMap, inbox: inbox, aether: aether}
}

// fillNameMap fetches Kind 0 metadata for authors not in nameMap and returns updated map.
func fillNameMap(events []nostr.Event, nameMap map[string]string) map[string]string {
	needNames := make(map[string]bool)
	for _, ev := range events {
		if _, ok := nameMap[ev.PubKey]; !ok || nameMap[ev.PubKey] == "" {
			needNames[ev.PubKey] = true
		}
	}
	if len(needNames) == 0 {
		return nameMap
	}
	authors := make([]string, 0, len(needNames))
	for pk := range needNames {
		authors = append(authors, pk)
	}
	filters := nostr.Filters{{
		Authors: authors,
		Kinds:   []int{nostr.KindSetMetadata},
		Limit:   len(authors),
	}}
	_, metaCh := pool.Sub(filters)
	ch := iterEventsWithTimeout(nostr.Unique(metaCh), 2*time.Second)
	for ev := range ch {
		if ev.Kind != nostr.KindSetMetadata {
			continue
		}
		var meta Metadata
		if err := json.Unmarshal([]byte(ev.Content), &meta); err != nil {
			continue
		}
		if meta.Name != "" {
			nameMap[ev.PubKey] = meta.Name
		}
	}
	return nameMap
}

// loadOurReactions fetches our kind 6/7 events for the given targets, returns likedMap and boostedMap.
func loadOurReactions(events []nostr.Event) (likedMap, boostedMap map[string]string) {
	likedMap = make(map[string]string)
	boostedMap = make(map[string]string)
	if config.PrivateKey == "" {
		return likedMap, boostedMap
	}
	pubkey := getPubKey(config.PrivateKey)
	targetIDs := make(map[string]bool)
	for _, ev := range events {
		targetIDs[ev.ID] = true
	}
	if len(targetIDs) == 0 {
		return likedMap, boostedMap
	}
	filters := nostr.Filters{{
		Authors: []string{pubkey},
		Kinds:   []int{nostr.KindReaction, nostr.KindBoost},
		Limit:   200,
	}}
	_, ch := pool.Sub(filters)
	for ev := range iterEventsWithTimeout(nostr.Unique(ch), 2*time.Second) {
		var targetID string
		for _, tag := range ev.Tags {
			if len(tag) > 0 && tag[0] == "e" && len(tag) > 1 {
				targetID = tag[1]
				break
			}
		}
		if targetID == "" || !targetIDs[targetID] {
			continue
		}
		if ev.Kind == nostr.KindReaction && ev.Content == "+" {
			likedMap[targetID] = ev.ID
		} else if ev.Kind == nostr.KindBoost {
			boostedMap[targetID] = ev.ID
		}
	}
	return likedMap, boostedMap
}

func loadFeedCmd(inbox, notesOnly, aether bool) tea.Cmd {
	return func() tea.Msg {
		return loadHomeFeed(inbox, notesOnly, aether)
	}
}
