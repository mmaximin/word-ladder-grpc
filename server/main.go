package main

import (
	"context"
	"log"
	"net"
	"strings"
	"word-ladder/dictionary"
	"word-ladder/game"
	"word-ladder/validation"

	pb "word-ladder/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWordLadderServiceServer
	game       *game.Game
	dictionary *dictionary.Dictionary
	factory    *game.GameFactory
}

func (s *server) StartGame(ctx context.Context, req *pb.StartGameRequest) (*pb.StartGameResponse, error) {
	s.game = s.factory.CreateGame()

	return &pb.StartGameResponse{
		StartWord:  s.game.StartWord,
		TargetWord: s.game.TargetWord,
	}, nil
}

func (s *server) SubmitWord(ctx context.Context, req *pb.SubmitWordRequest) (*pb.SubmitWordResponse, error) {
	word := strings.ToUpper(strings.TrimSpace(req.GetWord()))

	lengthValidator := validation.LengthValidator{}
	oneLetterValidator := validation.OneLetterValidator{}
	usedWordValidator := validation.UsedWordValidator{}
	dictionaryValidator := validation.DictionaryValidator{
		Dictionary: s.dictionary,
	}

	if !lengthValidator.Validate(s.game.CurrentWord, word, s.game) {
		return &pb.SubmitWordResponse{
			Valid:        false,
			Message:      "Invalid word. Incorrect length.",
			CurrentWord:  s.game.CurrentWord,
			TargetWord:   s.game.TargetWord,
			GameComplete: false,
		}, nil
	}

	if !dictionaryValidator.Validate(s.game.CurrentWord, word, s.game) {
		return &pb.SubmitWordResponse{
			Valid:        false,
			Message:      "Invalid word. Word not found in dictionary.",
			CurrentWord:  s.game.CurrentWord,
			TargetWord:   s.game.TargetWord,
			GameComplete: false,
		}, nil
	}

	if !oneLetterValidator.Validate(s.game.CurrentWord, word, s.game) {
		return &pb.SubmitWordResponse{
			Valid:        false,
			Message:      "Invalid word. More than one letter has changed.",
			CurrentWord:  s.game.CurrentWord,
			TargetWord:   s.game.TargetWord,
			GameComplete: false,
		}, nil
	}

	if !usedWordValidator.Validate(s.game.CurrentWord, word, s.game) {
		return &pb.SubmitWordResponse{
			Valid:        false,
			Message:      "Invalid word. Word already used.",
			CurrentWord:  s.game.CurrentWord,
			TargetWord:   s.game.TargetWord,
			GameComplete: false,
		}, nil
	}

	if word == s.game.TargetWord && len(s.game.UsedWords)+1 < 5 {
		return &pb.SubmitWordResponse{
			Valid:        false,
			Message:      "Invalid word. The ladder must contain at least 5 words.",
			CurrentWord:  s.game.CurrentWord,
			TargetWord:   s.game.TargetWord,
			GameComplete: false,
		}, nil
	}

	s.game.UsedWords[word] = true
	s.game.CurrentWord = word

	gameComplete := s.game.CurrentWord == s.game.TargetWord

	return &pb.SubmitWordResponse{
		Valid:        true,
		Message:      "Valid Word",
		CurrentWord:  s.game.CurrentWord,
		TargetWord:   s.game.TargetWord,
		GameComplete: gameComplete,
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	factory := game.NewGameFactory()

	s := &server{
		dictionary: dictionary.GetDictionary(),
		factory:    factory,
	}
	grpcServer := grpc.NewServer()
	pb.RegisterWordLadderServiceServer(grpcServer, s)

	log.Printf("server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
