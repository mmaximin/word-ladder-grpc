package game

import "math/rand"

type GamePair struct {
	Start  string
	Target string
}

type GameFactory struct {
	pairs []GamePair
}

func NewGameFactory() *GameFactory {
	return &GameFactory{
		pairs: []GamePair{
			{Start: "COLD", Target: "WARM"},
			{Start: "HEAD", Target: "TAIL"},
			{Start: "LOVE", Target: "HATE"},
		},
	}
}

// CreateGame creates a new game with a random word pair.
// The game ID and the username are given by the server.
func (f *GameFactory) CreateGame(id int32, username string) *Game {
	pair := f.pairs[rand.Intn(len(f.pairs))]

	return NewGame(id, username, pair.Start, pair.Target)
}
