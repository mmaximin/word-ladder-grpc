package validation

import (
	"word-ladder/dictionary"
	"word-ladder/game"
)

type DictionaryValidator struct {
	Dictionary *dictionary.Dictionary
}

func (v DictionaryValidator) Validate(currentWord string, newWord string, g *game.Game) bool {
	return v.Dictionary.Contains(newWord)
}
