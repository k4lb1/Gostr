package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

// Gostr: Gopher (RFC 1436) output constants.
const (
	gopherHost = "localhost"
	gopherPort = "70"
)

// ASCII art for Gostr header (Gopher type 'i' = info lines).
var gostrArt = []string{
	"      ________  ________   ________  _________  ________     ",
	"   /  _____/ /  __  __\\/   ____/ /___   ___/|   __   \\    ",
	"  /   \\  ___/  /  /  / \\____   \\    /  /    |  |__/  /    ",
	"  \\    \\_\\  \\  \\__/  / /       /   /  /     |      _/     ",
	"   \\________/\\______/ /_______/   /__/      |__|\\__\\      ",
	"                                                          v0.1.1",
	"         --- NOSTR LIKE GOPHER | GOSTR ---",
}

// printGostrHeader prints the Gostr ASCII art as Gopher info lines (type 'i').
// RFC 1436: i<display>\t<selector>\t<host>\t<port>; for info lines selector/host/port are empty.
func printGostrHeader() {
	for _, line := range gostrArt {
		fmt.Printf("i%s\t\t\t\r\n", line)
	}
}

// formatAsGopher converts a Nostr event into one or more Gopher directory lines.
// Kind 1 (Text Note): type '0' (text file). Display "[Author] Content", selector = event ID.
// Kind 0 (Profile): type '1' (directory). Display "Profile: <name>", selector = event ID.
// Other kinds are skipped (empty slice). Newlines in content are replaced with spaces.
func formatAsGopher(evt nostr.Event, nick *string) []string {
	author := shorten(evt.PubKey)
	if nick != nil && *nick != "" {
		author = *nick
	}

	switch evt.Kind {
	case nostr.KindTextNote:
		content := strings.ReplaceAll(evt.Content, "\n", " ")
		content = strings.ReplaceAll(content, "\t", " ")
		display := fmt.Sprintf("[%s] %s", author, content)
		return []string{fmt.Sprintf("0%s\t%s\t%s\t%s", display, evt.ID, gopherHost, gopherPort)}
	case nostr.KindSetMetadata:
		display := "Profile: " + author
		if evt.Content != "" {
			var meta Metadata
			if err := json.Unmarshal([]byte(evt.Content), &meta); err == nil && meta.Name != "" {
				display = "Profile: " + meta.Name
			}
		}
		return []string{fmt.Sprintf("1%s\t%s\t%s\t%s", display, evt.ID, gopherHost, gopherPort)}
	default:
		return nil
	}
}
