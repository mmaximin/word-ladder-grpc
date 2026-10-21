package validation

import "word-ladder/game"

type LengthValidator struct {
}

func (v LengthValidator) Validate(currentWord string, newWord string, g *game.Game) bool {
	return len(currentWord) == len(newWord)
}
