package validation

import "word-ladder/game"

type Validator interface {
	Validate(currentWord string, newWord string, g *game.Game) bool
}
