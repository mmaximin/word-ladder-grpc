package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	address := os.Getenv("STATS_SERVER_ADDR")
	if address == "" {
		address = "localhost:9000"
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatalf("could not connect to stats server: %v", err)
	}
	defer conn.Close()

	fmt.Printf("Connected to stats server at %s\n", address)
	fmt.Println("Commands: history <username>, leaderboard, quit")

	input := bufio.NewReader(os.Stdin)
	server := bufio.NewReader(conn)

	for {
		fmt.Print("stats > ")
		command, err := input.ReadString('\n')
		if err != nil {
			return
		}
		command = strings.TrimSpace(command)

		if command == "" {
			continue
		}

		_, err = fmt.Fprintf(conn, "%s\n", command)
		if err != nil {
			log.Fatalf("could not send command: %v", err)
		}

		for {
			line, err := server.ReadString('\n')
			if err != nil {
				log.Fatalf("connection closed by server: %v", err)
			}

			line = strings.TrimRight(line, "\r\n")
			if line == "END" {
				break
			}

			fmt.Println(line)
		}

		if strings.ToLower(command) == "quit" {
			return
		}
	}
}
