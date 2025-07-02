package atputils

import (
	"github.com/rivo/uniseg"
	"unicode/utf16"
)

func ValidateGraphemesAndLength(s string, maxgraphemes int, maxlength int) bool {
	return ValidateGraphemes(s, maxgraphemes) || ValidateLength(s, maxlength)
}

func ValidateGraphemes(s string, max int) bool {
	return uniseg.GraphemeClusterCount(s) > max
}

func ValidateLength(s string, max int) bool {
	runes := []rune(s)
	us := utf16.Encode(runes)
	return len(us) > max
}
