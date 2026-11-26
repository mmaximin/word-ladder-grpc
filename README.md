# Word Ladder – Assignment 2 (Multiplayer, RabbitMQ and Stats)

## Project Overview

This project is a distributed implementation of the Word Ladder game developed in Go.

The player starts with one word and has to reach a target word by changing one letter at a time. Each word submitted must be a valid English word, must have the correct length, must differ from the current word by exactly one letter and cannot have already been used in the same game.

Assignment 1 was a single-player client-server game using gRPC. In Assignment 2 the system has been extended so that:

- many players can play **at the same time**, each one with a username and their own game;
- each game counts the number of **turns** (submitted guesses);
- a player can **quit** (give up);
- when a game finishes, the game server publishes a **game event to RabbitMQ**;
- a separate **stats server** consumes those events, stores them in **SQLite** and answers questions from **stats clients** over plain **TCP sockets**;
- the server components run in **Docker** using **Docker Compose**.

---

## Assignment 2 Architecture

```
 +--------------+   gRPC    +----------------+   JSON event   +-----------+
 | game client  | --------> |  game-server   | -------------> | RabbitMQ  |
 | (player 1)   |           |  (gRPC, :50051)|  queue:        | (:5672,   |
 +--------------+           |  map of games  |  game_events   |  :15672)  |
 +--------------+   gRPC    |                |                +-----------+
 | game client  | --------> |                |                      |
 | (player 2)   |           +----------------+                      | consume
 +--------------+                                                   v
                                                           +----------------+
 +--------------+   TCP socket (text, one command/line)    |  stats-server  |
 | stats client | <--------------------------------------> |  (:9000)       |
 +--------------+                                          |  SQLite DB     |
 +--------------+   TCP socket                             |  game_results  |
 | stats client | <--------------------------------------> |                |
 +--------------+                                          +----------------+
```

- **Game client** (`client/`) – asks for a username, starts a game and sends words with gRPC.
- **Game server** (`server/`) – multiplayer gRPC server. Keeps all active games and publishes a RabbitMQ event when a game finishes.
- **RabbitMQ** – message broker between the game server and the stats server (queue `game_events`).
- **Stats server** (`statsserver/`) – consumes events from RabbitMQ, saves them in SQLite, and serves stats clients over TCP sockets.
- **Stats client** (`statsclient/`) – simple terminal program that sends commands to the stats server over a TCP socket.

The game server and the stats server never talk to each other directly. They only communicate through RabbitMQ, so the stats server can be stopped and restarted without affecting the games (the events wait in the queue).

---

## Game Client

`client/main.go`

1. Asks the player for a **username**.
2. Calls `StartGame(username)` and receives the **game ID**, start word and target word.
3. Reads words from the keyboard and calls `SubmitWord(game_id, word)` for each one.
4. Shows whether the word was valid, the current word, the target word and the number of turns.
5. If the player types `quit`, the client calls `QuitGame(game_id)` and the game is recorded as a **LOSS**.
6. When the target word is reached, the game is a **WIN** and the client ends.

The server address can be changed with the `GAME_SERVER_ADDR` environment variable (default `localhost:50051`).

---

## Multiplayer gRPC Game Server

`server/main.go`, `proto/wordladder.proto`

The service now has three remote procedures:

| RPC | Request | Response |
|---|---|---|
| `StartGame` | `username` | `game_id`, `start_word`, `target_word` |
| `SubmitWord` | `game_id`, `word` | `valid`, `message`, `current_word`, `target_word`, `game_complete`, `turns` |
| `QuitGame` | `game_id` | `message`, `turns` |

### How multiple simultaneous games work

In Assignment 1 the server stored only one game (`game *game.Game`). Now it stores a map of games:

```go
mu     sync.Mutex
games  map[int32]*game.Game
nextID int32
```

- `StartGame` locks the mutex, increases `nextID` to get a **unique game ID**, asks the `GameFactory` to create a game for that username, and stores it in the map.
- `SubmitWord` and `QuitGame` use the `game_id` from the request to find the correct game in the map. Each `Game` has its own current word, used words and turn counter, so the games are completely independent.
- gRPC runs every request in its own goroutine, so two players can send requests at the same moment. The `sync.Mutex` makes sure only one goroutine reads or changes the map and the games at a time.
- When a game finishes (WIN or LOSS) it is removed from the map and an event is published to RabbitMQ.

### Turns

Every word submitted with `SubmitWord` counts as one turn (valid or invalid), and the value is stored in the `Turns` field of the game.

---

## RabbitMQ and Game Events

`server/publisher.go`, `events/`

When a game finishes, the game server publishes a JSON message to the RabbitMQ queue **`game_events`**:

```json
{"username":"maria","game_id":1,"result":"WIN","turns":5}
```

- `result` is `WIN` when the target word is reached and `LOSS` when the player quits.
- The message is sent to the default exchange with the queue name as the routing key, so no extra exchanges or routing are needed.
- The queue is durable and the messages are persistent.
- The `GameEvent` struct and the queue name are in the shared `events/` package, so the publisher and the consumer always use the same format.
- The connection URL comes from the `RABBITMQ_URL` environment variable (default `amqp://guest:guest@localhost:5672/`; in Docker `amqp://guest:guest@rabbitmq:5672/`).
- Because RabbitMQ takes a few seconds to start, both servers retry the connection several times.

The RabbitMQ management website is available at **http://localhost:15672** (user `guest`, password `guest`). Open **Queues → game_events** to see the queue and the message rates.

---

## Stats Server

`statsserver/main.go`, `statsserver/consumer.go`, `statsserver/database.go`

The stats server is a completely separate program. It does two things at the same time:

1. **A goroutine consumes RabbitMQ events** (`ConsumeEvents`). Each event is saved in SQLite and only then acknowledged, so no event is lost if saving fails.
2. **The main goroutine accepts TCP socket clients.** Every client gets its own goroutine (`go handleClient(conn, db)`), so many stats clients can be connected at once.

### SQLite

The database file is `game_stats.db` (in Docker `/data/game_stats.db`, stored in a Docker volume so the data survives restarts). The SQLite driver is `modernc.org/sqlite`, which is written in pure Go, so no C compiler is needed.

```sql
CREATE TABLE IF NOT EXISTS game_results (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id   INTEGER NOT NULL,
    username  TEXT    NOT NULL,
    result    TEXT    NOT NULL,
    turns     INTEGER NOT NULL,
    played_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

History query:

```sql
SELECT game_id, result, turns FROM game_results WHERE username = ? ORDER BY id;
```

Leaderboard query (wins descending, then fewer average turns first):

```sql
SELECT username, COUNT(*) AS wins, AVG(turns) AS avg_turns
FROM game_results
WHERE result = 'WIN'
GROUP BY username
ORDER BY wins DESC, avg_turns ASC;
```

---

## Socket Communication

The stats server uses normal TCP sockets (`net.Listen` / `net.Dial`), **not gRPC**, on port **9000**.

The protocol is plain text:

- the client sends **one command per line**;
- the server answers with one or more lines, followed by a line containing only **`END`**, so the client knows the answer is complete.

### Available commands

| Command | Description |
|---|---|
| `history <username>` | All the games of that user (game ID, result, turns) |
| `leaderboard` | Wins and average turns per win of every user |
| `quit` | Close the connection |

Example:

```
stats > history aoife
Game 17 WIN 4 turns
Game 21 LOSS 2 turns
Game 24 WIN 6 turns

stats > leaderboard
User            Wins  Average turns per win
mohammad        4     5.25
aoife           2     5.00
ben             1     4.00
```

---

## Stats Client

`statsclient/main.go`

Connects to the stats server (`STATS_SERVER_ADDR`, default `localhost:9000`), reads commands from the keyboard, sends them over the socket and prints the answer until `END`. You can open as many stats clients as you want at the same time.

---

## Docker / Docker Compose

- `server/Dockerfile` – builds the game server image (includes `words_dictionary.json`).
- `statsserver/Dockerfile` – builds the stats server image.
- `docker-compose.yml` – runs three containers:

| Service | Image | Ports |
|---|---|---|
| `rabbitmq` | `rabbitmq:3-management` | 5672 (AMQP), 15672 (management website) |
| `game-server` | built from `server/Dockerfile` | 50051 (gRPC) |
| `stats-server` | built from `statsserver/Dockerfile` | 9000 (TCP) |

Inside Docker the services use service names instead of `localhost` (for example `RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/`). The game server and stats server wait until RabbitMQ is healthy before starting.

The clients are run on the host machine and connect to the published ports.

---

## How to Run the Complete System

Requirements: Go and Docker Desktop.

1. Start the servers (from the project root):

   ```
   docker compose up --build
   ```

2. In a new terminal, start a game client (repeat in another terminal for a second player):

   ```
   go run ./client
   ```

3. In a new terminal, start a stats client (repeat in another terminal for a second one):

   ```
   go run ./statsclient
   ```

4. Open http://localhost:15672 (guest / guest) to see RabbitMQ.

5. See the server logs:

   ```
   docker compose logs -f game-server stats-server
   ```

6. Stop everything:

   ```
   docker compose down          (keeps the database)
   docker compose down -v       (also deletes the database volume)
   ```

### Running without Docker (optional)

```
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
go run ./server
go run ./statsserver
go run ./client
go run ./statsclient
```

### Regenerating the gRPC code

After changing `proto/wordladder.proto`:

```
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/wordladder.proto
```

---

## Example Ladders

The `GameFactory` picks one of these pairs at random. The ladder must contain at least five words.

| Pair | Example ladder |
|---|---|
| COLD → WARM | COLD, CORD, CARD, WARD, WARM |
| HEAD → TAIL | HEAD, HEAL, TEAL, TELL, TALL, TAIL |
| LOVE → HATE | LOVE, ROVE, ROTE, RATE, HATE |

---

## Directory Structure

```
word-ladder/
├── client/
│   └── main.go              game client (gRPC)
├── server/
│   ├── main.go              multiplayer gRPC game server
│   ├── publisher.go         RabbitMQ publisher
│   └── Dockerfile
├── statsserver/
│   ├── main.go              TCP socket server + starts the consumer
│   ├── consumer.go          RabbitMQ consumer
│   ├── database.go          SQLite table and queries
│   └── Dockerfile
├── statsclient/
│   └── main.go              TCP socket stats client
├── events/
│   ├── event.go             GameEvent struct + queue name (shared)
│   └── rabbitmq.go          connect with retries + declare queue (shared)
├── proto/
│   ├── wordladder.proto
│   ├── wordladder.pb.go         (generated)
│   └── wordladder_grpc.pb.go    (generated)
├── game/
│   ├── game.go              Game state (ID, username, turns, ...)
│   └── factory.go           GameFactory
├── dictionary/
│   └── dictionary.go        Singleton dictionary
├── validation/              validation strategies
│   ├── validator.go
│   ├── length.go
│   ├── dictionary.go
│   ├── oneletter.go
│   └── usedword.go
├── words_dictionary.json
├── docker-compose.yml
├── .dockerignore
├── go.mod
├── go.sum
└── README.md
```

---

## Design Patterns

### 1. Singleton Pattern

**Location:** `dictionary/dictionary.go`

The game only needs one dictionary instance, so `words_dictionary.json` is loaded only once and the same instance is reused by all the games.

In Assignment 2 the server handles many requests at the same time, so the Singleton now uses `sync.Once`. This guarantees the dictionary is loaded exactly once even if two goroutines ask for it at the same moment.

### 2. Strategy Pattern

**Location:** `validation/`

All validators implement the same `Validator` interface, but each one checks a different rule:

- `LengthValidator` – the new word has the correct length.
- `DictionaryValidator` – the word exists in the dictionary.
- `OneLetterValidator` – exactly one letter has changed.
- `UsedWordValidator` – the word has not already been used in this game.

The validators did not need to change for Assignment 2, because they already receive the `*game.Game` they are checking. Each request simply passes its own game. The server runs them in `validateWord`.

### 3. Factory Pattern

**Location:** `game/factory.go`

`GameFactory` creates new games by choosing one of the known word pairs (COLD → WARM, HEAD → TAIL, LOVE → HATE).

In Assignment 2 `CreateGame(id, username)` also receives the game ID and the username, so every game created belongs to a player and has a unique ID.

---

## Game Validation

When the player submits a word, the server checks:

1. The word has the correct length.
2. The word exists in the English dictionary.
3. Exactly one letter has changed from the current word.
4. The word has not already been used.
5. The ladder contains at least five words before reaching the target.
6. Whether the new word is the target word.

Words are converted to uppercase, so `cord`, `Cord` and `CORD` are treated the same way.
