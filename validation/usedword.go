package validation

import "word-ladder/game"

type UsedWordValidator struct {
}

func (v UsedWordValidator) Validate(currentWord string, newWord string, g *game.Game) bool {
	return !g.UsedWords[newWord]
}
