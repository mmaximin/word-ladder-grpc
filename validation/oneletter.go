package validation

import "word-ladder/game"

type OneLetterValidator struct {
}

func (v OneLetterValidator) Validate(currentWord string, newWord string, g *game.Game) bool {
	if len(currentWord) != len(newWord) {
		return false
	}

	differences := 0

	for i := 0; i < len(currentWord); i++ {
		if currentWord[i] != newWord[i] {
			differences++
		}
	}

	return differences == 1
}
