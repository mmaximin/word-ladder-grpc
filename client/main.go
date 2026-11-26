package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "word-ladder/proto"

	"google.golang.org/grpc"
)

func main() {
	address := os.Getenv("GAME_SERVER_ADDR")
	if address == "" {
		address = "localhost:50051"
	}

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewWordLadderServiceClient(conn)

	reader := bufio.NewReader(os.Stdin)

	// Ask for the username before starting the game.
	username := ""
	for username == "" {
		fmt.Print("Enter your username > ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		username = strings.TrimSpace(line)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cr, err := c.StartGame(ctx, &pb.StartGameRequest{Username: username})

	if err != nil {
		log.Fatalf("could not start game: %v", err)
	}

	gameID := cr.GetGameId()

	log.Printf("\nWORD LADDER!\nPlayer: %s\nGame ID: %d\nStart word: %s\nTarget word: %s\n(type 'quit' to give up)",
		username,
		gameID,
		cr.GetStartWord(),
		cr.GetTargetWord(),
	)

	for {
		fmt.Print("Enter word > ")
		word, err := reader.ReadString('\n')
		if err != nil {
			// Input closed (for example Ctrl+D): treat it as giving up.
			word = "quit"
		}
		word = strings.TrimSpace(word)

		if word == "" {
			continue
		}

		// The player gives up: the server records the game as a LOSS.
		if strings.ToLower(word) == "quit" {
			qr, err := c.QuitGame(
				context.Background(),
				&pb.QuitGameRequest{
					GameId: gameID,
				},
			)

			if err != nil {
				log.Fatalf("could not quit game: %v", err)
			}

			log.Printf("%s\nTurns: %d", qr.GetMessage(), qr.GetTurns())
			break
		}

		sr, err := c.SubmitWord(
			context.Background(),
			&pb.SubmitWordRequest{
				GameId: gameID,
				Word:   word,
			},
		)

		if err != nil {
			log.Fatalf("could not submit word: %v", err)
		}

		log.Printf(
			"%s\nCurrent word: %s\nTarget word: %s\nTurns: %d",
			sr.GetMessage(),
			sr.GetCurrentWord(),
			sr.GetTargetWord(),
			sr.GetTurns(),
		)

		if sr.GetGameComplete() {
			log.Printf("Word Ladder complete! You won in %d turns.", sr.GetTurns())
			break
		}
	}
}
