package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"word-ladder/dictionary"
	"word-ladder/events"
	"word-ladder/game"
	"word-ladder/validation"

	pb "word-ladder/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWordLadderServiceServer
	dictionary *dictionary.Dictionary
	factory    *game.GameFactory
	publisher  *Publisher

	// All the active games, using the game ID as the key.
	// The mutex protects the map and the games inside it, because
	// gRPC runs every request in its own goroutine.
	mu     sync.Mutex
	games  map[int32]*game.Game
	nextID int32
}

func (s *server) StartGame(ctx context.Context, req *pb.StartGameRequest) (*pb.StartGameResponse, error) {
	username := strings.TrimSpace(req.GetUsername())
	if username == "" {
		return nil, fmt.Errorf("a username is required")
	}

	s.mu.Lock()
	s.nextID++
	g := s.factory.CreateGame(s.nextID, username)
	s.games[g.ID] = g
	s.mu.Unlock()

	log.Printf("game %d started for %s: %s -> %s", g.ID, g.Username, g.StartWord, g.TargetWord)

	return &pb.StartGameResponse{
		GameId:     g.ID,
		StartWord:  g.StartWord,
		TargetWord: g.TargetWord,
	}, nil
}

// validateWord runs all the validation strategies on the word.
// It returns an empty string if the word is valid, or the error message.
func (s *server) validateWord(g *game.Game, word string) string {
	lengthValidator := validation.LengthValidator{}
	oneLetterValidator := validation.OneLetterValidator{}
	usedWordValidator := validation.UsedWordValidator{}
	dictionaryValidator := validation.DictionaryValidator{
		Dictionary: s.dictionary,
	}

	if !lengthValidator.Validate(g.CurrentWord, word, g) {
		return "Invalid word. Incorrect length."
	}

	if !dictionaryValidator.Validate(g.CurrentWord, word, g) {
		return "Invalid word. Word not found in dictionary."
	}

	if !oneLetterValidator.Validate(g.CurrentWord, word, g) {
		return "Invalid word. More than one letter has changed."
	}

	if !usedWordValidator.Validate(g.CurrentWord, word, g) {
		return "Invalid word. Word already used."
	}

	if word == g.TargetWord && len(g.UsedWords)+1 < 5 {
		return "Invalid word. The ladder must contain at least 5 words."
	}

	return ""
}

func (s *server) SubmitWord(ctx context.Context, req *pb.SubmitWordRequest) (*pb.SubmitWordResponse, error) {
	word := strings.ToUpper(strings.TrimSpace(req.GetWord()))

	s.mu.Lock()

	g, ok := s.games[req.GetGameId()]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("game %d not found", req.GetGameId())
	}

	// Every submitted guess counts as one turn, valid or not.
	g.Turns++

	errorMessage := s.validateWord(g, word)
	if errorMessage != "" {
		response := &pb.SubmitWordResponse{
			Valid:        false,
			Message:      errorMessage,
			CurrentWord:  g.CurrentWord,
			TargetWord:   g.TargetWord,
			GameComplete: false,
			Turns:        g.Turns,
		}
		s.mu.Unlock()
		return response, nil
	}

	g.UsedWords[word] = true
	g.CurrentWord = word

	gameComplete := g.CurrentWord == g.TargetWord

	response := &pb.SubmitWordResponse{
		Valid:        true,
		Message:      "Valid Word",
		CurrentWord:  g.CurrentWord,
		TargetWord:   g.TargetWord,
		GameComplete: gameComplete,
		Turns:        g.Turns,
	}

	// A finished game is removed from the active games.
	if gameComplete {
		delete(s.games, g.ID)
	}

	s.mu.Unlock()

	if gameComplete {
		log.Printf("game %d finished: %s WIN in %d turns", g.ID, g.Username, g.Turns)
		s.publisher.Publish(events.GameEvent{
			Username: g.Username,
			GameID:   g.ID,
			Result:   "WIN",
			Turns:    g.Turns,
		})
	}

	return response, nil
}

func (s *server) QuitGame(ctx context.Context, req *pb.QuitGameRequest) (*pb.QuitGameResponse, error) {
	s.mu.Lock()

	g, ok := s.games[req.GetGameId()]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("game %d not found", req.GetGameId())
	}

	delete(s.games, g.ID)

	s.mu.Unlock()

	log.Printf("game %d finished: %s LOSS (quit) after %d turns", g.ID, g.Username, g.Turns)
	s.publisher.Publish(events.GameEvent{
		Username: g.Username,
		GameID:   g.ID,
		Result:   "LOSS",
		Turns:    g.Turns,
	})

	return &pb.QuitGameResponse{
		Message: "You gave up. The target word was " + g.TargetWord + ".",
		Turns:   g.Turns,
	}, nil
}

// getEnv returns the environment variable, or a default value if it is not set.
func getEnv(name string, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	port := getEnv("GAME_SERVER_PORT", "50051")
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	publisher := NewPublisher(rabbitURL)
	defer publisher.Close()

	factory := game.NewGameFactory()

	s := &server{
		dictionary: dictionary.GetDictionary(),
		factory:    factory,
		publisher:  publisher,
		games:      make(map[int32]*game.Game),
		nextID:     0,
	}
	grpcServer := grpc.NewServer()
	pb.RegisterWordLadderServiceServer(grpcServer, s)

	log.Printf("server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
