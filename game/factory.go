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

func (f *GameFactory) CreateGame(id int32, username string) *Game {
	pair := f.pairs[rand.Intn(len(f.pairs))]

	return NewGame(id, username, pair.Start, pair.Target)
}
