package main

import (
	"regexp"

	"github.com/TheZoraiz/ascii-image-converter/aic_package"
)

const asciiImageMaxWidth = 60

var imageURLRegex = regexp.MustCompile(`(?i)https?://[^\s"'<>]+\.(png|jpg|jpeg|webp|gif)(?:\?[^\s"'<>]*)?`)

// extractImageURLs finds image URLs in note content (Nostr notes often embed image links).
func extractImageURLs(content string) []string {
	matches := imageURLRegex.FindAllString(content, -1)
	seen := make(map[string]bool)
	var out []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// fetchAndConvertToASCII downloads an image from URL and converts to ASCII art.
// maxWidth limits output width in characters to avoid huge 4K images.
func fetchAndConvertToASCII(url string, maxWidth int) (string, error) {
	if maxWidth <= 0 {
		maxWidth = asciiImageMaxWidth
	}
	flags := aic_package.DefaultFlags()
	flags.Width = maxWidth
	return aic_package.Convert(url, flags)
}
