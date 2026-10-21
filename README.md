# Word Ladder

## Project Overview

This project is a distributed implementation of the Word Ladder game developed in Go.

The player starts with one word and has to reach a target word by changing one letter at a time. Each word submitted must be a valid English word, must have the correct length, must differ from the current word by exactly one letter and cannot have already been used in the same game.

The application follows a client-server architecture. The client communicates with the server using gRPC and Protocol Buffers. The server stores the game state and checks whether each word entered by the player is valid.

---

## Architecture

The project is divided into different packages so that each part of the application has a clear responsibility:

- `client/` - Handles the interaction with the player and sends requests to the server.
- `server/` - Implements the gRPC service and coordinates the game logic.
- `proto/` - Contains the Protocol Buffers file and the generated gRPC code.
- `game/` - Stores the game state and contains the game creation logic.
- `dictionary/` - Loads and provides access to the English word dictionary.
- `validation/` - Contains the different validation rules used by the game.

This structure helps to keep the project organised and separates the different responsibilities of the application.

---

## gRPC

Communication between the client and server is implemented using gRPC.

The service is defined in:

`proto/wordladder.proto`

The service has two remote procedures:

### StartGame

Starts a new game using the `GameFactory`.

The factory selects one of the available start and target word pairs and creates a new `Game`.

The server then sends the starting word and target word back to the client.

### SubmitWord

Sends the word entered by the player to the server.

The server checks whether the word is valid and sends a response back to the client containing:

- whether the word is valid;
- a message explaining the result;
- the current word;
- the target word;
- whether the game has been completed.

Protocol Buffers are used to define the request and response messages exchanged between the client and server.

---

## Design Patterns

### 1. Singleton Pattern

**Location:** `dictionary/dictionary.go`

The Singleton pattern is used for the dictionary.

The game only needs one dictionary instance, so the `words_dictionary.json` file is loaded only once when the dictionary is first requested.

After that, the same dictionary instance is reused by the server.

I used this pattern because the dictionary file is large and there is no reason to load the same information several times.

---

### 2. Strategy Pattern

**Location:** `validation/`

The Strategy pattern is used for the different validation rules.

All validators follow the same `Validator` interface, but each one is responsible for a different check.

The validation strategies are:

- `LengthValidator` - checks that the new word has the correct length.
- `DictionaryValidator` - checks that the word exists in the dictionary.
- `OneLetterValidator` - checks that exactly one letter has changed.
- `UsedWordValidator` - checks that the word has not already been used in the game.

I used this pattern because it keeps each validation rule separate. This makes the code easier to understand and also makes it easier to add another validation rule in the future without changing the existing validators.

---

### 3. Factory Pattern

**Location:** `game/factory.go`

The Factory pattern is used to create new Word Ladder games.

`GameFactory` contains a list of known start and target word pairs.

The current pairs are:

- COLD -> WARM
- HEAD -> TAIL
- LOVE -> HATE

When the client calls `StartGame`, the server asks the factory to create a new game. The factory selects one of these pairs and returns a new `Game`.

I used this pattern so that the server does not need to decide directly which words should be used when creating a game.

It also makes it easy to add more Word Ladder pairs later.

---

## Game Validation

When the player submits a word, the server checks:

1. The word has the correct length.
2. The word exists in the English dictionary.
3. Exactly one letter has changed from the current word.
4. The word has not already been used.
5. The ladder contains at least five words before reaching the target.
6. Whether the new word is the target word.

If all the required checks are valid, the current word is updated and the player can continue.

When the target word is reached, the game is completed.

---

## Dictionary

The project uses `words_dictionary.json` as the source of valid English words.

The dictionary is loaded when the Singleton instance is first created and is reused while the server is running.

The words entered by the player are converted to uppercase before validation so that inputs such as `cord`, `Cord` and `CORD` can be handled in the same way.

---

## Running the Project

First, start the gRPC server:

`go run ./server`

The server listens on port `50051`.

Then open another terminal and start the client:

`go run ./client`

The client starts a new game, displays the starting word and target word and asks the player to enter words.

The player can continue entering words until the target word is reached.

Starting the client again creates a new game through the `StartGame` remote procedure.

---

## Example

An example valid Word Ladder is:

COLD  
CORD  
CARD  
WARD  
WARM

Each word differs from the previous word by exactly one letter and all the words exist in the dictionary.

When `WARM` is reached, the game is complete.