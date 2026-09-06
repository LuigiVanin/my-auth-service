package utils

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MacroCase builds the identifier half of a scoped profile key, as in
// `<org_uuid>:GESTAO_DE_VENDAS`. Every run that is not ASCII alphanumeric becomes one
// separator, `:` included - it would otherwise collide with the scope separator.
//
// Returns "" when the name reduces to nothing, which the caller answers with a 400.
func MacroCase(text string) string {
	words := []string{}
	current := strings.Builder{}

	for _, symbol := range norm.NFD.String(text) {
		// What is left of an accent once the rune was decomposed.
		if unicode.Is(unicode.Mn, symbol) {
			continue
		}

		if isMacroSymbol(symbol) {
			current.WriteRune(unicode.ToUpper(symbol))
			continue
		}

		if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return strings.Join(words, "_")
}

func isMacroSymbol(symbol rune) bool {
	switch {
	case symbol >= 'a' && symbol <= 'z':
		return true
	case symbol >= 'A' && symbol <= 'Z':
		return true
	case symbol >= '0' && symbol <= '9':
		return true
	}

	return false
}
