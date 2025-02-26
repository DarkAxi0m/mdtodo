package main

import (
	"strings"
"unicode/utf8")


//Thankyou robots
func isEmojiStart(s string) bool {
	if s == "" {
		return false
	}

	r, _ := utf8.DecodeRuneInString(s)

	// Check if the rune falls within known emoji ranges
	switch {
	case r >= 0x1F600 && r <= 0x1F64F: // Emoticons
		return true
	case r >= 0x1F300 && r <= 0x1F5FF: // Miscellaneous Symbols & Pictographs
		return true
	case r >= 0x1F680 && r <= 0x1F6FF: // Transport & Map Symbols
		return true
	case r >= 0x2600 && r <= 0x26FF:   // Miscellaneous Symbols
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF: // Regional Indicator Symbols (Flags)
		return true
	default:
		return false
	}
}


func extractEmoji(s string) (string, string) {
	if s == "" || !isEmojiStart(s) {
		return "", s // No emoji found
	}

	_, size := utf8.DecodeRuneInString(s) // Get the first rune
	return s[:size], strings.TrimSpace(s[size:])             // Split into emoji and remainder
}

