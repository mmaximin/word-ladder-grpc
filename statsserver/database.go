package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"word-ladder/events"

	_ "modernc.org/sqlite" // SQLite driver (pure Go, no C compiler needed)
)

func OpenDatabase(path string) *sql.DB {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("could not open database: %v", err)
	}

	// SQLite only allows one writer at a time, so we use a single
	// connection. database/sql makes the goroutines take turns.
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS game_results (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id   INTEGER NOT NULL,
			username  TEXT    NOT NULL,
			result    TEXT    NOT NULL,
			turns     INTEGER NOT NULL,
			played_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		log.Fatalf("could not create table: %v", err)
	}

	log.Printf("SQLite database ready at %s", path)
	return db
}

func SaveResult(db *sql.DB, event events.GameEvent) error {
	_, err := db.Exec(
		"INSERT INTO game_results (game_id, username, result, turns) VALUES (?, ?, ?, ?)",
		event.GameID, event.Username, event.Result, event.Turns,
	)
	return err
}

// History returns one line per game played by the user.
func History(db *sql.DB, username string) ([]string, error) {
	rows, err := db.Query(
		"SELECT game_id, result, turns FROM game_results WHERE username = ? ORDER BY id",
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var gameID, turns int
		var result string

		if err := rows.Scan(&gameID, &result, &turns); err != nil {
			return nil, err
		}

		lines = append(lines, fmt.Sprintf("Game %d %s %d turns", gameID, result, turns))
	}

	if len(lines) == 0 {
		lines = append(lines, "No games found for "+username)
	}

	return lines, rows.Err()
}

// Leaderboard returns the number of wins and the average turns per win
// for every user with at least one win, best players first.
func Leaderboard(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT username, COUNT(*) AS wins, AVG(turns) AS avg_turns
		FROM game_results
		WHERE result = 'WIN'
		GROUP BY username
		ORDER BY wins DESC, avg_turns ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := []string{fmt.Sprintf("%-15s %-5s %s", "User", "Wins", "Average turns per win")}
	for rows.Next() {
		var username string
		var wins int
		var avgTurns float64

		if err := rows.Scan(&username, &wins, &avgTurns); err != nil {
			return nil, err
		}

		lines = append(lines, fmt.Sprintf("%-15s %-5d %.2f", username, wins, avgTurns))
	}

	if len(lines) == 1 {
		lines = append(lines, "No wins recorded yet")
	}

	return lines, rows.Err()
}

// cleanUsername is used so that "Maria" and "maria " are the same user.
func cleanUsername(username string) string {
	return strings.TrimSpace(username)
}
