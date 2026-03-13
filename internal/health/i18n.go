package health

// LangStrings maps translation keys to localised text for one language.
type LangStrings map[string]string

// langs holds all supported translations. "en" is also the fallback.
var langs = map[string]LangStrings{
	"en": en,
	"ru": ru,
	"sr": sr,
}

// GetStrings returns localised strings for the given lang code (falls back to "en").
func GetStrings(lang string) LangStrings {
	if ls, ok := langs[lang]; ok {
		return ls
	}
	return en
}
