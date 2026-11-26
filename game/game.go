package game

type Game struct {
	ID          int32
	Username    string
	StartWord   string
	CurrentWord string
	TargetWord  string
	UsedWords   map[string]bool
	Turns       int32
}

func NewGame(id int32, username string, startWord string, targetWord string) *Game {
	return &Game{
		ID:          id,
		Username:    username,
		StartWord:   startWord,
		CurrentWord: startWord,
		TargetWord:  targetWord,
		UsedWords: map[string]bool{
			startWord: true,
		},
		Turns: 0,
	}
}
