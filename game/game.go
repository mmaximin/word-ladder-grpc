package game

type Game struct {
	StartWord   string
	CurrentWord string
	TargetWord  string
	UsedWords   map[string]bool
}

func NewGame(startWord string, targetWord string) *Game {
	return &Game{
		StartWord:   startWord,
		CurrentWord: startWord,
		TargetWord:  targetWord,
		UsedWords: map[string]bool{
			startWord: true,
		},
	}
}
