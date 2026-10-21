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
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewWordLadderServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cr, err := c.StartGame(ctx, &pb.StartGameRequest{})

	if err != nil {
		log.Fatalf("could not start game: %v", err)
	}

	log.Printf("\nWORD LADDER!\nStart word: %s\nTarget word: %s",
		cr.GetStartWord(),
		cr.GetTargetWord(),
	)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter word > ")
		word, _ := reader.ReadString('\n')
		word = strings.TrimSpace(word)

		sr, err := c.SubmitWord(
			context.Background(),
			&pb.SubmitWordRequest{
				Word: word,
			},
		)

		if err != nil {
			log.Fatalf("could not submit word: %v", err)
		}

		log.Printf(
			"%s\nCurrent word: %s\nTarget word: %s",
			sr.GetMessage(),
			sr.GetCurrentWord(),
			sr.GetTargetWord(),
		)

		if sr.GetGameComplete() {
			log.Println("Word Ladder complete!")
			break
		}
	}
}
