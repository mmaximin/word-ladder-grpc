package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

// getEnv returns the environment variable, or a default value if it is not set.
func getEnv(name string, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// handleClient serves one stats client. Each client runs in its own goroutine.
//
// Protocol: the client sends one command per line. The server answers with
// one or more lines and then a line containing only "END".
func handleClient(conn net.Conn, db *sql.DB) {
	defer conn.Close()

	log.Printf("stats client connected: %s", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	writer := bufio.NewWriter(conn)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(line)

		if len(parts) == 0 {
			continue
		}

		command := strings.ToLower(parts[0])
		log.Printf("%s: %s", conn.RemoteAddr(), line)

		var lines []string
		var err error

		switch {
		case command == "history" && len(parts) == 2:
			lines, err = History(db, cleanUsername(parts[1]))
		case command == "history":
			lines = []string{"Usage: history <username>"}
		case command == "leaderboard":
			lines, err = Leaderboard(db)
		case command == "quit":
			writer.WriteString("Bye\nEND\n")
			writer.Flush()
			log.Printf("stats client disconnected: %s", conn.RemoteAddr())
			return
		default:
			lines = []string{"Unknown command. Use: history <username>, leaderboard, quit"}
		}

		if err != nil {
			lines = []string{"Error: " + err.Error()}
		}

		for _, l := range lines {
			writer.WriteString(l + "\n")
		}
		writer.WriteString("END\n")
		writer.Flush()
	}

	log.Printf("stats client disconnected: %s", conn.RemoteAddr())
}

func main() {
	port := getEnv("STATS_SERVER_PORT", "9000")
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	dbPath := getEnv("DB_PATH", "game_stats.db")

	db := OpenDatabase(dbPath)
	defer db.Close()

	// 1. Consume RabbitMQ events in the background.
	go ConsumeEvents(rabbitURL, db)

	// 2. Accept TCP socket clients at the same time.
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	fmt.Printf("stats server listening on port %s\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		// One goroutine per client, so many clients can be served at once.
		go handleClient(conn, db)
	}
}
