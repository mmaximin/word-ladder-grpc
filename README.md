# Word Ladder – Assignment 2

This is my Word Ladder game from Assignment 1, extended for Assignment 2.

Quick reminder of the game: you start with a word and have to get to a target word by changing one letter at a time. Every word has to be a real English word, the same length, and not used before. The ladder also needs at least 5 words.

In Assignment 1 only one person could play. Now:

- lots of people can play at the same time, each with their own username and game
- the server counts your turns
- you can type `quit` to give up
- when a game ends, the result gets sent to RabbitMQ
- a separate stats server picks up the results, saves them in SQLite and lets you check stats over a TCP socket
- the servers run in Docker

---

## How it all fits together

```
game client  --gRPC-->  game-server  --JSON-->  RabbitMQ  -->  stats-server (SQLite)
game client  --gRPC-->                                              ^
                                                                    |
stats client  <------------------- TCP socket --------------------->|
stats client  <------------------- TCP socket --------------------->|
```

The game server and the stats server never talk to each other directly. Everything goes through RabbitMQ. So if the stats server is down for a bit, the messages just wait in the queue until it comes back.

---

## The parts

### Game client (`client/`)

- asks for your username
- starts a game and shows your game ID, start word and target word
- you type words, and it shows if they're valid and how many turns you've used
- type `quit` to give up (that counts as a LOSS)
- reach the target word and you WIN

### Game server (`server/`)

This is the gRPC server. It has 3 calls:

| RPC | You send | You get back |
|---|---|---|
| `StartGame` | username | game ID, start word, target word |
| `SubmitWord` | game ID, word | valid or not, message, current word, target word, game complete, turns |
| `QuitGame` | game ID | message, turns |

**How multiple games work:** before, the server only had one `game`. Now it has a map of games, with the game ID as the key:

```go
mu     sync.Mutex
games  map[int32]*game.Game
nextID int32
```

- `StartGame` gives each new game the next ID, so IDs never repeat
- `SubmitWord` and `QuitGame` use the game ID to find the right game
- every game has its own words and turn count, so games don't mess with each other
- gRPC handles each request in its own goroutine, so the mutex stops two requests from changing the map at the same time
- when a game ends, it's removed from the map and an event is sent to RabbitMQ

Every word you submit counts as a turn, even the invalid ones.

### RabbitMQ (`server/publisher.go`, `events/`)

When a game ends, the game server sends a message like this to a queue called `game_events`:

```json
{"username":"maria","game_id":1,"result":"WIN","turns":5}
```

`result` is `WIN` if you reached the target and `LOSS` if you quit.

The `events/` folder has the message struct and the queue name. Both servers use it, so they always agree on the format. Both servers also retry the connection a few times, because RabbitMQ takes a few seconds to start.

You can see RabbitMQ in the browser at http://localhost:15672 (user `guest`, password `guest`). Go to **Queues**, then `game_events`.

### Stats server (`statsserver/`)

This is a separate program that does two things at once:

1. a goroutine reads events from RabbitMQ and saves them in SQLite
2. the main part accepts TCP connections, and each client gets its own goroutine, so lots of clients can be connected at once

The database has one table:

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

The leaderboard query:

```sql
SELECT username, COUNT(*) AS wins, AVG(turns) AS avg_turns
FROM game_results
WHERE result = 'WIN'
GROUP BY username
ORDER BY wins DESC, avg_turns ASC;
```

I used `modernc.org/sqlite` because it's pure Go, so it works on Windows and in Docker without installing a C compiler.

### Stats client (`statsclient/`)

It connects to the stats server on port 9000 using a plain TCP socket (no gRPC here). You type a command, it sends it, and it prints the answer. The server sends `END` at the end of each answer, so the client knows when to stop reading.

Commands:

| Command | What it does |
|---|---|
| `history <username>` | shows all the games of that user |
| `leaderboard` | shows wins and average turns per win for everyone |
| `quit` | disconnects |

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

## Docker

- `server/Dockerfile` builds the game server (it also copies the dictionary)
- `statsserver/Dockerfile` builds the stats server
- `docker-compose.yml` starts everything:

| Service | Port |
|---|---|
| `rabbitmq` (`rabbitmq:3-management`) | 5672 and 15672 (website) |
| `game-server` | 50051 |
| `stats-server` | 9000 |

Inside Docker the services find each other by name. For example, the servers connect to `amqp://guest:guest@rabbitmq:5672/` instead of `localhost`. The SQLite file is kept in a Docker volume, so your stats stay there after a restart.

The clients aren't in Docker. You run them in normal terminals.

---

## How to run it

You need Go and Docker Desktop (it has to be open).

**1. Start the servers** (from the project folder):

```
docker compose up --build
```

Wait until you see `server listening` and `waiting for game events`.

**2. Play** (one terminal per player):

```
go run ./client
```

**3. Check stats** (one terminal per stats client):

```
go run ./statsclient
```

**4. Look at RabbitMQ:** go to http://localhost:15672 (guest / guest).

**Stop everything:**

```
docker compose down
```

or `docker compose down -v` if you also want to delete the stats.

### Ladders that work

The game picks one of these at random:

| Start → Target | Words to type |
|---|---|
| COLD → WARM | cord, card, ward, warm |
| HEAD → TAIL | heal, teal, tell, tall, tail |
| LOVE → HATE | rove, rote, rate, hate |

### If you change the .proto file

```
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/wordladder.proto
```

### Running without Docker (if you want)

```
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
go run ./server
go run ./statsserver
go run ./client
go run ./statsclient
```

---

## Folder structure

```
word-ladder/
├── client/         game client
├── server/         game server + RabbitMQ publisher + Dockerfile
├── statsserver/    stats server (RabbitMQ consumer, SQLite, TCP) + Dockerfile
├── statsclient/    stats client
├── events/         event struct + RabbitMQ helpers (used by both servers)
├── proto/          .proto file + generated gRPC code
├── game/           Game struct + GameFactory
├── dictionary/     dictionary (Singleton)
├── validation/     validation rules (Strategy)
├── words_dictionary.json
├── docker-compose.yml
├── go.mod / go.sum
└── README.md
```

---

## Design patterns (from Assignment 1, still here)

**Singleton (`dictionary/dictionary.go`):** the dictionary file is big, so it only gets loaded once and every game shares it. I changed it to use `sync.Once`, so it's still loaded only once even if several requests ask for it at the same time.

**Strategy (`validation/`):** each rule (length, dictionary, one letter, used word) is its own validator, and they all have the same `Validate` function. I didn't need to change anything here, because each validator already gets the game it's checking.

**Factory (`game/factory.go`):** `GameFactory` picks a random word pair and creates the game. Now it also gets the game ID and the username, so every game belongs to a player.
